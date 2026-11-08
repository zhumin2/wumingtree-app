package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/httpx"
)

// RequireAdmin 管理端点角色守卫（在 Auth 之后使用）。授权在 service 层兜底。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRole)
		if r, _ := role.(string); r != model.RoleAdmin {
			httpx.Fail(c, code.ErrNoPermission)
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	}
}
