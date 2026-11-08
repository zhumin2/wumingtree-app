package module

import (
	"github.com/gin-gonic/gin"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/httpx"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

// 幂等令牌：客户端生成，重复请求返回首次结果（架构设计 3.4，先于频控）。
const clientTokenHeader = "client_token"

// RegisterHollow 树洞端点（组内各路由自带 Auth 中间件）。
func RegisterHollow(rg *gin.RouterGroup, hollow *service.HollowService, report *service.ReportService, authMW, adminMW gin.HandlerFunc) {
	posts := rg.Group("/posts", authMW)

	// 发帖（默认匿名；机审拒绝级 4101；危机词不拦截 + 求助卡标记）
	posts.POST("", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		var req struct {
			Mood        string  `json:"mood" binding:"required"`
			Title       string  `json:"title" binding:"required"`
			Body        *string `json:"body"`
			IsAnonymous *bool   `json:"isAnonymous"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		isAnon := true
		if req.IsAnonymous != nil {
			isAnon = *req.IsAnonymous
		}
		vo, err := hollow.CreatePost(c.Request.Context(), id, req.Mood, req.Title, req.Body, isAnon, c.GetHeader(clientTokenHeader))
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, vo)
	})

	// 树洞列表：心情筛选 + 时间倒序（红线：无热度排序）
	posts.GET("", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		page, err := service.Page{}, error(nil)
		page.Cursor = c.Query("cursor")
		page.Limit = atoiDefault(c.Query("limit"), 20)
		vo, err := hollow.ListPosts(c.Request.Context(), id, c.Query("mood"), page)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, vo)
	})

	// 详情 + 首屏楼层（含危机标记）
	posts.GET("/:id", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		postID, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		vo, err := hollow.GetPost(c.Request.Context(), id, postID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, vo)
	})

	// 楼层分页
	posts.GET("/:id/floors", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		postID, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		page, _ := service.Page{}, error(nil)
		page.Cursor = c.Query("cursor")
		page.Limit = atoiDefault(c.Query("limit"), 20)
		floors, next, err := hollow.ListFloors(c.Request.Context(), id, postID, page)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": floors, "nextCursor": next})
	})

	// 发表楼层（可对楼层回应）
	posts.POST("/:id/floors", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		postID, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		var req struct {
			Content       string `json:"content" binding:"required"`
			ParentFloorID *int64 `json:"parentFloorId"`
			IsAnonymous   *bool  `json:"isAnonymous"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		isAnon := true
		if req.IsAnonymous != nil {
			isAnon = *req.IsAnonymous
		}
		vo, err := hollow.CreateFloor(c.Request.Context(), id, postID, req.Content, req.ParentFloorID, isAnon, c.GetHeader(clientTokenHeader))
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, vo)
	})

	// 帖子级回应 toggle（hug 抱抱 / me_too 我也是；无负向类型——红线）
	posts.POST("/:id/reactions", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		postID, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		var req struct {
			Type string `json:"type" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		vo, err := hollow.TogglePostReaction(c.Request.Context(), id, postID, req.Type)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, vo)
	})

	// 删帖（回应一并隐藏，不可恢复）
	posts.DELETE("/:id", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		postID, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		if err := hollow.DeletePost(c.Request.Context(), id, postID); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, nil)
	})

	// 楼层级回应 toggle（hug 抱一下 / helpful 有帮助）
	floors := rg.Group("/floors", authMW)
	floors.POST("/:id/reactions", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		floorID, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		var req struct {
			Type string `json:"type" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		vo, err := hollow.ToggleFloorReaction(c.Request.Context(), id, floorID, req.Type)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, vo)
	})

	// 举报（reporter 身份任何接口不回传——红线）
	rg.Group("/reports", authMW).POST("", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		var req struct {
			TargetType string `json:"targetType" binding:"required"`
			TargetID   int64  `json:"targetId" binding:"required"`
			Reason     string `json:"reason" binding:"required"`
			Detail     string `json:"detail"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		if err := report.Create(c.Request.Context(), id, req.TargetType, req.TargetID, req.Reason, req.Detail); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, nil)
	})

	// 我的树洞（含匿名帖；服务端按作者过滤——红线）
	rg.Group("/me/posts", authMW).GET("", func(c *gin.Context) {
		id, ok := memberIDOf(c)
		if !ok {
			httpx.Fail(c, code.ErrUnauthorized)
			return
		}
		page, _ := service.Page{}, error(nil)
		page.Cursor = c.Query("cursor")
		page.Limit = atoiDefault(c.Query("limit"), 20)
		vo, err := hollow.MyPosts(c.Request.Context(), id, page)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, vo)
	})

	// 管理台：举报队列与处理
	admin := rg.Group("/admin", authMW, adminMW)
	admin.GET("/reports", func(c *gin.Context) {
		offset := atoiDefault(c.Query("offset"), 0)
		limit := atoiDefault(c.Query("limit"), 20)
		list, total, err := report.List(c.Request.Context(), c.Query("status"), offset, limit)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": list, "total": total})
	})
	admin.POST("/reports/:id/handle", func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		adminID, _ := memberIDOf(c)
		var req struct {
			Action string `json:"action" binding:"required"`
			Note   string `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		if err := report.Handle(c.Request.Context(), id, adminID, req.Action, req.Note); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, nil)
	})
}
