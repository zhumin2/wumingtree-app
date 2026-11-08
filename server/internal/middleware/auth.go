package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/httpx"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/jwt"
)

const (
	CtxMemberID = "member_id"
	CtxRole     = "role"
)

// Auth 只做认证（解析 Bearer access token 并注入成员身份），不做授权；
// 授权（成员状态 / 报名关系 / 内容归属）在 service 层完成（架构设计 3.5）。
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(raw, "Bearer ")
		if !ok || token == "" {
			httpx.Fail(c, code.ErrUnauthorized)
			c.AbortWithStatus(http.StatusOK)
			return
		}
		claims, err := jwt.Parse(secret, token)
		if err != nil {
			httpx.Fail(c, code.ErrUnauthorized)
			c.AbortWithStatus(http.StatusOK)
			return
		}
		if claims.Typ != jwt.TypAccess {
			httpx.Fail(c, code.ErrUnauthorized)
			c.AbortWithStatus(http.StatusOK)
			return
		}
		mid, perr := parseSubject(claims.Subject)
		if perr != nil {
			httpx.Fail(c, code.ErrUnauthorized)
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Set(CtxMemberID, mid)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}
