package service

import (
	"context"
	"time"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/ratelimit"
)

// ReportCreator 举报落库。
type ReportCreator interface {
	Create(rep *model.Report) error
}

// ReportLister 管理台举报队列。
type ReportLister interface {
	ListByStatus(status string, offset, limit int) ([]model.Report, int64, error)
}

// ReportHandler 处理举报（remove_content 联动下架在 repository 事务内）。
type ReportHandler interface {
	Handle(reportID, adminID int64, action, note string) error
}

// ReportService 举报（reporter_id 内部保留，任何接口不回传——红线）。
type ReportService struct {
	reports ReportCreator
	lister  ReportLister
	handler ReportHandler
	limiter ratelimit.Limiter
}

func NewReportService(reports ReportCreator, lister ReportLister, handler ReportHandler, limiter ratelimit.Limiter) *ReportService {
	return &ReportService{reports: reports, lister: lister, handler: handler, limiter: limiter}
}

// 举报原因枚举（产品红线：隐私泄露 / 评判攻击 / 广告引流 / 其他）。
func validReportReason(r string) bool {
	switch r {
	case "privacy", "attack", "ads", "other":
		return true
	}
	return false
}

func validReportTarget(t string) bool {
	switch t {
	case "post", "floor":
		return true // message / resource_file 随对应模块开放
	}
	return false
}

// Create 提交举报（20 条/天；被举报人不会知道是谁举报的——红线）。
func (s *ReportService) Create(ctx context.Context, memberID int64, targetType string, targetID int64, reason, detail string) error {
	if !validReportTarget(targetType) || !validReportReason(reason) || len([]rune(detail)) > 200 || targetID <= 0 {
		return code.ErrValidation
	}
	if ok, _ := s.limiter.Allow(ctx, "report:day:"+int64Key(memberID), 20, 24*time.Hour); !ok {
		return code.ErrRateLimited
	}
	rep := model.Report{
		TargetType: targetType,
		TargetID:   targetID,
		Reason:     reason,
		ReporterID: memberID,
		Status:     model.ReportOpen,
	}
	if detail != "" {
		d := detail
		rep.Detail = &d
	}
	return s.reports.Create(&rep)
}

// ReportVO 管理台队列视图。红线：不含 reporter_id。
type ReportVO struct {
	ID         int64             `json:"id"`
	TargetType string            `json:"targetType"`
	TargetID   int64             `json:"targetId"`
	Reason     string            `json:"reason"`
	Detail     string            `json:"detail,omitempty"`
	Status     string            `json:"status"`
	HandledBy  int64             `json:"handledBy,omitempty"`
	Action     string            `json:"action,omitempty"`
	Note       string            `json:"note,omitempty"`
	CreatedAt  string            `json:"-"`
	Extra      map[string]string `json:"-"`
}

// List 管理台队列（offset 分页）。
func (s *ReportService) List(ctx context.Context, status string, offset, limit int) ([]model.Report, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.lister.ListByStatus(status, offset, limit)
}

// 处理动作枚举（细化设计 2.4）。
const (
	ReportActionRemove = "remove_content"
	ReportActionNone   = "no_action"
)

// Handle 处理举报：remove_content（下架目标）/ no_action（必须附说明）。
func (s *ReportService) Handle(ctx context.Context, reportID, adminID int64, action, note string) error {
	if action != ReportActionRemove && action != ReportActionNone {
		return code.ErrValidation
	}
	if action == ReportActionNone && trimSpaces(note) == "" {
		return code.ErrValidation // 无处置必须附说明
	}
	if len([]rune(note)) > 200 {
		return code.ErrValidation
	}
	return s.handler.Handle(reportID, adminID, action, note)
}
