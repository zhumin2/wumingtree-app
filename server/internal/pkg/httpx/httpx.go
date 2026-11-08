// Package httpx 统一响应信封。
//
// 铁律（《技术架构设计》3.2）：任何返回必须走 OK / Fail，禁止裸 c.JSON——
// 客户端解析器只认信封，漏一处就是线上 bug。CI 的 envelope-guard 会拦截违规。
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
)

const requestIDKey = "request_id"

// Envelope 全接口统一信封：{"code":0,"message":"ok","data":...,"request_id":"..."}
type Envelope struct {
	Code      int64  `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

// SetRequestID 注入请求上下文（由 middleware.RequestID 调用）。
func SetRequestID(c *gin.Context, id string) {
	c.Set(requestIDKey, id)
	c.Header("X-Request-Id", id)
}

func requestID(c *gin.Context) string {
	v, _ := c.Get(requestIDKey)
	s, _ := v.(string)
	return s
}

// OK 成功信封。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: code.CodeOK, Message: "ok", Data: data, RequestID: requestID(c)})
}

// Fail 业务失败信封：按哨兵错误映射 code；未知错误归 9000 且不泄露内部信息。
func Fail(c *gin.Context, err error) {
	bc, msg := code.Of(err)
	c.JSON(http.StatusOK, Envelope{Code: bc, Message: msg, Data: nil, RequestID: requestID(c)})
}

// FailWith 业务失败且 data 携带附加信息（如 1001 的逐字段 errors）。
func FailWith(c *gin.Context, err error, data any) {
	bc, msg := code.Of(err)
	c.JSON(http.StatusOK, Envelope{Code: bc, Message: msg, Data: data, RequestID: requestID(c)})
}

// FailCode 直接按码失败（module 层少数无 service 依赖的入口使用）。
func FailCode(c *gin.Context, codeNum int64, message string) {
	c.JSON(http.StatusOK, Envelope{Code: codeNum, Message: message, Data: nil, RequestID: requestID(c)})
}
