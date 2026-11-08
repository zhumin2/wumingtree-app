package module

import (
	"github.com/gin-gonic/gin"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/httpx"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

// RegisterAuth 准入端点：验证码、登录、刷新（公开）。
// 契约初稿的 GET /applications/mine 由 login outcome 承担（申请者无会话）；
// 通知接入（M5）后按契约评审结论调整。
func RegisterAuth(rg *gin.RouterGroup, auth *service.AuthService) {
	g := rg.Group("/auth")

	g.POST("/sms-codes", func(c *gin.Context) {
		var req struct {
			Phone string `json:"phone" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		if err := auth.SendCode(c.Request.Context(), req.Phone, c.ClientIP()); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, nil)
	})

	g.POST("/login", func(c *gin.Context) {
		var req struct {
			Phone string `json:"phone" binding:"required"`
			Code  string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		res, err := auth.Login(c.Request.Context(), req.Phone, req.Code, c.ClientIP())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, res)
	})

	g.POST("/refresh", func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refreshToken" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		tokens, err := auth.Refresh(c.Request.Context(), req.RefreshToken)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, tokens)
	})
}
