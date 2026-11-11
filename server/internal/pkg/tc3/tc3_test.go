package tc3

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// 官方文档签名示例固定向量（https://cloud.tencent.com/document/api/213/30654）：
// SecretKey 在文档中打码，但派生密钥链（kDate/kService/kSigning）与最终签名完整给出，
// 足以校验除「TC3+SecretKey → kDate」之外的每一个签名步骤；
// kDate 那一步由 kDateVectorTest 用独立实现（Python HMAC）预计算的值覆盖。
const (
	vecBody = `{"Limit": 1, "Filters": [{"Values": ["\u672a\u547d\u540d"], "Name": "instance-name"}]}`

	vecPayloadHash     = "35e9c5b0e3ae67532d3c9f17ead6c90222632e5b1ff7f6e89887f1398934f064"
	vecCanonicalHash   = "7019a55be8395899b900fb5564e4200d984910f34794a27cb3fb7d10ff6a1e84"
	vecKDateHex        = "da98fb70dcf6b112dc21038d1eeeb3a95c74b4dcb12c1131f864f6066bd02be0"
	vecKServiceHex     = "8d70cbefb03939f929db64d32dc2ba89b1095620119fe3e050e2b18c5bd2752f"
	vecKSigningHex     = "b596b923aad85185e2d1f6659d2a062e0a86731226e021e61bfe06f7ed05f5af"
	vecSignatureHex    = "10b1a37a7301a02ca19a647ad722d5e43b4b3cff309d421d85b46093f6ab6c4f"
	vecDate            = "2019-02-25"
	vecTimestamp       = "1551113065"
	vecService         = "cvm"
	vecHost            = "cvm.tencentcloudapi.com"
	vecAction          = "DescribeInstances"
	vecKDateTestKey    = "wumingtree-test-secret-key"
	vecKDateTestExpect = "2bbca96da2c366dda655a87b9dfd5d4076b8f271319a479fc4e79183eebed2b0"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex vector: %v", err)
	}
	return b
}

func TestPayloadHashVector(t *testing.T) {
	if got := sha256Hex([]byte(vecBody)); got != vecPayloadHash {
		t.Fatalf("payload hash = %s, want %s", got, vecPayloadHash)
	}
}

func TestCanonicalRequestVector(t *testing.T) {
	got := canonicalRequest(vecHost, vecAction, "application/json; charset=utf-8", vecPayloadHash)
	want := "POST\n/\n\n" +
		"content-type:application/json; charset=utf-8\n" +
		"host:cvm.tencentcloudapi.com\n" +
		"x-tc-action:describeinstances\n" +
		"\n" +
		"content-type;host;x-tc-action\n" +
		vecPayloadHash
	if got != want {
		t.Fatalf("canonical request mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
	if sha256Hex([]byte(got)) != vecCanonicalHash {
		t.Fatalf("canonical hash mismatch")
	}
}

func TestStringToSignVector(t *testing.T) {
	got := stringToSign(vecTimestamp, vecDate+"/"+vecService+"/tc3_request", vecCanonicalHash)
	want := "TC3-HMAC-SHA256\n" + vecTimestamp + "\n" + vecDate + "/" + vecService + "/tc3_request\n" + vecCanonicalHash
	if got != want {
		t.Fatalf("string to sign mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestDeriveSigningKeyVector(t *testing.T) {
	// 文档打码 SecretKey，但其 kDate 派生结果已知 → 从 kDate 起逐级校验。
	kDate := mustHex(t, vecKDateHex)
	kService := hmacSHA256(kDate, []byte(vecService))
	if !hmac.Equal(kService, mustHex(t, vecKServiceHex)) {
		t.Fatalf("kService mismatch: %s", hex.EncodeToString(kService))
	}
	kSigning := hmacSHA256(kService, []byte("tc3_request"))
	if !hmac.Equal(kSigning, mustHex(t, vecKSigningHex)) {
		t.Fatalf("kSigning mismatch: %s", hex.EncodeToString(kSigning))
	}
	sts := stringToSign(vecTimestamp, vecDate+"/"+vecService+"/tc3_request", vecCanonicalHash)
	sig := hmacSHA256(kSigning, []byte(sts))
	if !hmac.Equal(sig, mustHex(t, vecSignatureHex)) {
		t.Fatalf("signature mismatch: %s", hex.EncodeToString(sig))
	}
}

// kDateVectorTest 覆盖链路第一步（TC3+SecretKey → kDate），
// 期望值由独立实现（Python hmac）按同一向量预计算。
func TestKDateVector(t *testing.T) {
	kDate := hmacSHA256([]byte("TC3"+vecKDateTestKey), []byte(vecDate))
	if !hmac.Equal(kDate, mustHex(t, vecKDateTestExpect)) {
		t.Fatalf("kDate mismatch: %s", hex.EncodeToString(kDate))
	}
}

// TestDoSignsAndParses 端到端：测试端充当腾讯云服务端，
// 用收到的头重建签名逐字节校验，并验证 Response 信封解析。
func TestDoSignsAndParses(t *testing.T) {
	secretKey := "test-secret-key"
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Response": map[string]any{"Echo": "ok", "RequestId": "req-1"},
		})
	}))
	defer srv.Close()

	ts := time.Unix(1551113065, 0)
	cl := &Client{
		Cred:    Credential{SecretID: "test-id", SecretKey: secretKey},
		Host:    "sms.tencentcloudapi.com",
		Service: "sms",
		Region:  "ap-guangzhou",
		HTTP:    srv.Client(),
		Now:     func() time.Time { return ts },
		BaseURL: srv.URL, // 打到测试服务端；签名仍按真实 Host 计算
	}

	var out struct {
		Echo string `json:"Echo"`
	}
	if err := cl.Do(context.Background(), "SendSms", "2021-01-11", map[string]any{"a": 1}, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if out.Echo != "ok" {
		t.Fatalf("Echo = %q", out.Echo)
	}

	// 服务端视角重算签名（与 sign 相同的派生链，手写一遍防实现自洽）。
	body, _ := json.Marshal(map[string]any{"a": 1})
	date := ts.UTC().Format("2006-01-02")
	scope := date + "/sms/tc3_request"
	hashedCanonical := sha256.Sum256([]byte(
		"POST\n/\n\ncontent-type:application/json; charset=utf-8\nhost:sms.tencentcloudapi.com\n" +
			"x-tc-action:sendsms\n\ncontent-type;host;x-tc-action\n" + sha256Hex(body)))
	sts := "TC3-HMAC-SHA256\n" + strconv.FormatInt(ts.Unix(), 10) + "\n" + scope + "\n" + hex.EncodeToString(hashedCanonical[:])
	kDate := hmacSHA256([]byte("TC3"+secretKey), []byte(date))
	kSigning := hmacSHA256(hmacSHA256(kDate, []byte("sms")), []byte("tc3_request"))
	wantSig := hex.EncodeToString(hmacSHA256(kSigning, []byte(sts)))
	wantAuth := "TC3-HMAC-SHA256 Credential=test-id/" + scope +
		", SignedHeaders=content-type;host;x-tc-action, Signature=" + wantSig
	if gotAuth != wantAuth {
		t.Fatalf("authorization mismatch:\ngot:  %s\nwant: %s", gotAuth, wantAuth)
	}
}

func TestDoSurfacesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Response": map[string]any{
				"Error": map[string]any{"Code": "AuthFailure.SignatureFailure", "Message": "bad sig"},
			},
		})
	}))
	defer srv.Close()

	cl := &Client{Cred: Credential{"id", "key"}, Host: "sms.tencentcloudapi.com", Service: "sms",
		HTTP: srv.Client(), Now: time.Now, BaseURL: srv.URL}
	err := cl.Do(context.Background(), "SendSms", "2021-01-11", map[string]any{}, nil)
	if err == nil {
		t.Fatal("want error for API Error")
	}
}
