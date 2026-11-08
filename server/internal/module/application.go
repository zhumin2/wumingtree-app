package module

import (
	"github.com/gin-gonic/gin"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/httpx"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

// RegisterApplications 申请提交（公开）与管理端审核。
func RegisterApplications(rg *gin.RouterGroup, svc *service.ApplicationService, audit *service.AuditWriter, authMW, adminMW gin.HandlerFunc) {
	// 提交申请（公开；邀请码校验在事务内）
	rg.POST("/applications", func(c *gin.Context) {
		var req struct {
			InviteCode string `json:"inviteCode" binding:"required"`
			Nickname   string `json:"nickname" binding:"required"`
			Phone      string `json:"phone" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		id, _, err := svc.Submit(c.Request.Context(), req.InviteCode, req.Nickname, req.Phone)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		app, err := svc.GetApplication(id)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, applicationVO(app))
	})

	admin := rg.Group("/admin", authMW, adminMW)

	// 申请列表（?status=&offset=&limit=）
	admin.GET("/applications", func(c *gin.Context) {
		status := c.Query("status")
		offset, limit := pageParams(c)
		list, total, err := svc.ListApplications(status, offset, limit)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		vos := make([]*applicationVOPtr, 0, len(list))
		for i := range list {
			vos = append(vos, applicationVO(&list[i]))
		}
		httpx.OK(c, gin.H{"items": vos, "total": total})
	})

	// 通过：事务建成员；写审计
	admin.POST("/applications/:id/approve", func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		adminID, _ := memberIDOf(c)
		memberID, err := svc.Approve(c.Request.Context(), id, adminID, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"memberId": memberID})
	})

	// 拒绝：原因模板
	admin.POST("/applications/:id/reject", func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		var req struct {
			Reason string `json:"reason" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		adminID, _ := memberIDOf(c)
		if err := svc.Reject(c.Request.Context(), id, adminID, req.Reason, c.ClientIP(), c.Request.UserAgent()); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, nil)
	})

	// 手机号明文查看（独立端点 + phone.view 审计；架构设计 5.3/5.4）
	admin.GET("/members/:id/phone", func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			httpx.Fail(c, code.ErrValidation)
			return
		}
		adminID, _ := memberIDOf(c)
		phone, err := audit.PlainPhone(c.Request.Context(), id, adminID, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"phone": phone})
	})
}

func pageParams(c *gin.Context) (int, int) {
	offset := atoiDefault(c.Query("offset"), 0)
	limit := atoiDefault(c.Query("limit"), 20)
	return offset, limit
}

func atoiDefault(s string, def int) int {
	n := 0
	if s == "" {
		return def
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	return n
}

type applicationVOPtr = service.ApplicationVO

func applicationVO(a *model.Application) *service.ApplicationVO {
	v := &service.ApplicationVO{ID: a.ID, Status: a.Status, SubmittedAt: a.CreatedAt, ReviewedAt: a.ReviewedAt}
	if a.RejectReason != nil {
		v.RejectReason = *a.RejectReason
	}
	return v
}
