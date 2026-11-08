package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
)

func init() { gin.SetMode(gin.TestMode) }

func perform(h gin.HandlerFunc) (*httptest.ResponseRecorder, Envelope) {
	r := gin.New()
	r.Use(func(c *gin.Context) { SetRequestID(c, "req-test") })
	r.GET("/x", h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	var env Envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	return w, env
}

func TestOK_EnvelopeShape(t *testing.T) {
	w, env := perform(func(c *gin.Context) { OK(c, gin.H{"k": "v"}) })
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 恒为 200, got %d", w.Code)
	}
	if env.Code != 0 || env.Message != "ok" || env.RequestID != "req-test" {
		t.Fatalf("信封字段不符: %+v", env)
	}
}

func TestFail_UnknownErrorNoLeak(t *testing.T) {
	_, env := perform(func(c *gin.Context) { Fail(c, errors.New("内部堆栈: /opt/secret")) })
	if env.Code != code.CodeInternal {
		t.Fatalf("want 9000, got %d", env.Code)
	}
	if env.Message != code.ErrInternal.Message {
		t.Fatalf("未知错误不得泄露内部信息: %s", env.Message)
	}
}

func TestFail_WrappedSentinel(t *testing.T) {
	_, env := perform(func(c *gin.Context) {
		Fail(c, fmt.Errorf("ctx: %w", code.ErrQuotaFull))
	})
	if env.Code != code.CodeQuotaFull || env.Message != code.ErrQuotaFull.Message {
		t.Fatalf("wrapped sentinel 映射失败: %+v", env)
	}
}

func TestFailWith_CarriesData(t *testing.T) {
	_, env := perform(func(c *gin.Context) {
		FailWith(c, code.ErrValidation, gin.H{"errors": []gin.H{{"field": "consents"}}})
	})
	if env.Code != code.CodeValidation || env.Data == nil {
		t.Fatalf("FailWith 未携带 data: %+v", env)
	}
}

// TestNoRouteGoesThroughEnvelope 由 router 测试覆盖；此处保证 FailCode 可用。
func TestFailCode(t *testing.T) {
	_, env := perform(func(c *gin.Context) { FailCode(c, code.CodeNotFound, "接口不存在") })
	if env.Code != code.CodeNotFound {
		t.Fatalf("want %d, got %d", code.CodeNotFound, env.Code)
	}
}
