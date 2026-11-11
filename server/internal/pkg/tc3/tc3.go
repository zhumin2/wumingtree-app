// Package tc3 腾讯云 API v3 签名客户端（TC3-HMAC-SHA256）。
//
// 纯标准库实现，不引云厂商 SDK（架构设计 5.1：go.mod 保持轻量）。
// 签名算法与官方文档一致：https://cloud.tencent.com/document/api/213/30654。
// 固定向量测试用官方文档示例的派生密钥链（tc3_test.go），防签名回归。
package tc3

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Credential 腾讯云访问管理凭证（控制台「访问管理 → API 密钥管理」）。
type Credential struct {
	SecretID  string
	SecretKey string
}

// Client 单服务的签名 HTTP 客户端（sms / tms 各建一个）。
type Client struct {
	Cred    Credential
	Host    string // 如 sms.tencentcloudapi.com
	Service string // 如 sms
	Region  string // 如 ap-guangzhou；为空则不发送 X-TC-Region
	HTTP    *http.Client
	Now     func() time.Time // 可注入时钟（测试固定向量用）
	BaseURL string           // 覆盖请求基地址（仅测试注入 httptest 用；为空时 https://Host）
}

// New 构造客户端，默认 5s 超时（同步机审/发短信不阻塞请求太久）。
func New(cred Credential, host, service, region string) *Client {
	return &Client{
		Cred:    cred,
		Host:    host,
		Service: service,
		Region:  region,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
		Now:     time.Now,
	}
}

// Do 发起 POST JSON 调用：payload 序列化 → TC3 签名 → 解析 {"Response":{...}} 信封。
// 服务端返回 Error 时返回错误；out 为 nil 可忽略结果。
func (c *Client) Do(ctx context.Context, action, version string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("tc3: marshal payload: %w", err)
	}
	base := c.BaseURL
	if base == "" {
		base = "https://" + c.Host
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("tc3: new request: %w", err)
	}
	if err := c.sign(req, action, version, body); err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("tc3: do: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("tc3: read body: %w", err)
	}
	var env struct {
		Response json.RawMessage `json:"Response"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("tc3: bad envelope (http %d): %w", resp.StatusCode, err)
	}
	if len(env.Response) == 0 {
		return fmt.Errorf("tc3: empty Response (http %d)", resp.StatusCode)
	}
	var inner struct {
		Error *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	}
	if err := json.Unmarshal(env.Response, &inner); err != nil {
		return fmt.Errorf("tc3: bad response: %w", err)
	}
	if inner.Error != nil {
		return fmt.Errorf("tc3: %s %s: %s: %s", action, version, inner.Error.Code, inner.Error.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(env.Response, out); err != nil {
		return fmt.Errorf("tc3: decode result: %w", err)
	}
	return nil
}

// sign 按 TC3-HMAC-SHA256 规范为请求补齐签名头。
func (c *Client) sign(req *http.Request, action, version string, body []byte) error {
	ts := c.Now().UTC()
	date := ts.Format("2006-01-02")
	timestamp := strconv.FormatInt(ts.Unix(), 10)
	contentType := "application/json; charset=utf-8"

	hashedPayload := sha256Hex(body)
	hashedCanonical := sha256Hex([]byte(canonicalRequest(c.Host, action, contentType, hashedPayload)))
	scope := date + "/" + c.Service + "/tc3_request"
	sts := stringToSign(timestamp, scope, hashedCanonical)
	kSigning := deriveSigningKey(c.Cred.SecretKey, date, c.Service)
	signature := hex.EncodeToString(hmacSHA256(kSigning, []byte(sts)))

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Timestamp", timestamp)
	if c.Region != "" {
		req.Header.Set("X-TC-Region", c.Region)
	}
	req.Header.Set("Authorization",
		"TC3-HMAC-SHA256 Credential="+c.Cred.SecretID+"/"+scope+
			", SignedHeaders=content-type;host;x-tc-action, Signature="+signature)
	return nil
}

// canonicalRequest 规范请求串（POST /、空 query、三个签名头）。
func canonicalRequest(host, action, contentType, hashedPayload string) string {
	return strings.Join([]string{
		http.MethodPost,
		"/",
		"",
		"content-type:" + contentType,
		"host:" + host,
		"x-tc-action:" + strings.ToLower(action),
		"",
		"content-type;host;x-tc-action",
		hashedPayload,
	}, "\n")
}

// stringToSign 待签串。
func stringToSign(timestamp, scope, hashedCanonical string) string {
	return strings.Join([]string{
		"TC3-HMAC-SHA256",
		timestamp,
		scope,
		hashedCanonical,
	}, "\n")
}

// deriveSigningKey 派生签名密钥：kDate → kService → kSigning。
func deriveSigningKey(secretKey, date, service string) []byte {
	kDate := hmacSHA256([]byte("TC3"+secretKey), []byte(date))
	kService := hmacSHA256(kDate, []byte(service))
	return hmacSHA256(kService, []byte("tc3_request"))
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}
