package module

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/zhumin2/wumingtree-app/server/internal/middleware"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/httpx"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

func memberIDOf(c *gin.Context) (int64, bool) {
	v, ok := c.Get(middleware.CtxMemberID)
	id, ok2 := v.(int64)
	return id, ok && ok2
}

// RegisterMember 我的模块（组内已挂 Auth 中间件）。
func RegisterMember(rg *gin.RouterGroup, svc *service.MemberService) {
	g := rg.Group("/me")

	g.GET("", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		vo, err := svc.Me(c.Request.Context(), id)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, vo)
	})

	g.PATCH("", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		var req service.MemberPatch
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		if err := svc.UpdateMe(c.Request.Context(), id, req); err != nil {
			httpx.Fail(c, err)
			return
		}
		vo, err := svc.Me(c.Request.Context(), id)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, vo)
	})

	// 红点：仅布尔，无计数（红线）
	g.GET("/badges", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		httpx.OK(c, svc.Badges(c.Request.Context(), id))
	})

	g.GET("/summary", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		v, err := svc.Summary(c.Request.Context(), id)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, v)
	})
}

func pathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	return id, err == nil && id > 0
}
