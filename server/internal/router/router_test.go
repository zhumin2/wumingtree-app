package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/zhumin2/wumingtree-app/server/internal/config"
)

func testEngine(t *testing.T) *gin.Engine {
	t.Helper()
	return New(config.Config{Env: "dev", Addr: ":0"}, nil, nil)
}

func TestHealthz_Envelope(t *testing.T) {
	e := testEngine(t)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var env struct {
		Code int64 `json:"code"`
		Data struct {
			Status string `json:"status"`
			DB     bool   `json:"db"`
			Redis  bool   `json:"redis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("信封解析失败: %v", err)
	}
	if env.Code != 0 || env.Data.Status != "ok" || env.Data.DB || env.Data.Redis {
		t.Fatalf("无依赖时 healthz 应为 code=0 且 db/redis=false: %+v", env)
	}
}

func TestNoRoute_Envelope5001(t *testing.T) {
	// 客户端解析器只认信封：未匹配路由也必须走信封（架构设计 3.2 铁律）。
	e := testEngine(t)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/none", nil))
	var env struct {
		Code int64 `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("NoRoute 未走信封: %v", err)
	}
	if env.Code != 5001 {
		t.Fatalf("want 5001, got %d", env.Code)
	}
}
