package audit

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/tc3"
)

func newTencentAuditor(t *testing.T, handler http.HandlerFunc) (TencentAuditor, *int64) {
	t.Helper()
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	cl := tc3.New(tc3.Credential{SecretID: "id", SecretKey: "key"}, "tms.tencentcloudapi.com", "tms", "ap-guangzhou")
	cl.HTTP = srv.Client()
	cl.BaseURL = srv.URL
	return TencentAuditor{TC: cl}, &hits
}

func tmsResponse(suggestion, label string, keywords []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Response": map[string]any{
				"Suggestion": suggestion,
				"Label":      label,
				"Keywords":   keywords,
			},
		})
	}
}

func TestTencentAuditor_SuggestionMapping(t *testing.T) {
	cases := []struct {
		suggestion string
		want       string
	}{
		{"Pass", ActionAllow},
		{"Block", ActionReject},
		{"Review", ActionPending},
		{"Unexpected", ActionPending}, // 未知值按可疑处理
	}
	for _, c := range cases {
		a, _ := newTencentAuditor(t, tmsResponse(c.suggestion, "Abuse", []string{"加个微信吧"}))
		got := a.CheckText("一条普通内容")
		if got.Action != c.want {
			t.Fatalf("suggestion %s: action = %s, want %s", c.suggestion, got.Action, c.want)
		}
		if c.want != ActionAllow && got.Hit != "加个微信吧" {
			t.Fatalf("suggestion %s: hit = %s, want keywords", c.suggestion, got.Hit)
		}
	}
}

func TestTencentAuditor_RequestShape(t *testing.T) {
	var gotBody map[string]any
	a, _ := newTencentAuditor(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		tmsResponse("Pass", "Normal", nil)(w, r)
	})
	a.CheckText("正常分享")
	if gotBody["Content"] != base64.StdEncoding.EncodeToString([]byte("正常分享")) {
		t.Fatalf("Content not base64 of text: %v", gotBody["Content"])
	}
}

func TestTencentAuditor_CrisisShortCircuitsCloud(t *testing.T) {
	a, hits := newTencentAuditor(t, tmsResponse("Block", "Abuse", nil))
	got := a.CheckText("我现在真的不想活了")
	if got.Action != ActionAllow || !got.Crisis || got.Hit != "不想活" {
		t.Fatalf("crisis result = %+v, want allow+crisis", got)
	}
	if *hits != 0 {
		t.Fatalf("crisis branch must not call cloud, got %d calls", *hits)
	}
}

func TestTencentAuditor_LocalWordsBeforeCloud(t *testing.T) {
	a, hits := newTencentAuditor(t, tmsResponse("Pass", "Normal", nil))
	if got := a.CheckText("来我群里聊，加微信详谈"); got.Action != ActionReject {
		t.Fatalf("local reject word = %+v", got)
	}
	if *hits != 0 {
		t.Fatalf("local word list should short-circuit, got %d cloud calls", *hits)
	}
	if got := a.CheckText("你就是个废物"); got.Action != ActionPending {
		t.Fatalf("local pending word = %+v", got)
	}
}

func TestTencentAuditor_CloudErrorDegradesPending(t *testing.T) {
	a, _ := newTencentAuditor(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Response": map[string]any{
				"Error": map[string]any{"Code": "InternalError", "Message": "down"},
			},
		})
	})
	got := a.CheckText("一条普通内容")
	if got.Action != ActionPending {
		t.Fatalf("cloud error: action = %s, want pending", got.Action)
	}
}
