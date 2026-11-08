// Package middleware 只放认证与请求标记类横切件；授权判断在 service 层（架构设计 3.5）。
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/httpx"
)

// RequestID 为每个请求生成 ID 并回写响应头，与服务端日志、客户端 X-Request-Id 串联。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		httpx.SetRequestID(c, id)
		c.Next()
	}
}
