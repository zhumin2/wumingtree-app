package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

// ApplicationRepo 加入申请（含邀请码事务消费，细化设计 2.1）。
type ApplicationRepo struct{ DB *gorm.DB }

// Submit 事务内：锁定邀请码 → 校验并消费次数 → 防重 → 落申请（service.ApplicationSubmitter）。
func (r *ApplicationRepo) Submit(p service.SubmitParams) (int64, error) {
	now := time.Now()
	var appID int64
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var ic model.InviteCode
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code = ?", p.InviteCode).First(&ic).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return code.ErrInviteInvalid
		}
		if err != nil {
			return err
		}
		if ic.Status != model.InviteActive || ic.UsedCount >= ic.MaxUses ||
			now.Before(ic.ValidFrom) || now.After(ic.ValidUntil) {
			return code.ErrInviteInvalid
		}
		// 消耗次数；用尽即置 exhausted
		if err := tx.Model(&model.InviteCode{}).Where("id = ?", ic.ID).
			UpdateColumn("used_count", gorm.Expr("used_count+1")).Error; err != nil {
			return err
		}
		if ic.UsedCount+1 >= ic.MaxUses {
			if err := tx.Model(&model.InviteCode{}).Where("id = ?", ic.ID).
				UpdateColumn("status", model.InviteExhausted).Error; err != nil {
				return err
			}
		}
		// 防重：同摘要存在未结申请 → 4203
		var pending int64
		if err := tx.Model(&model.Application{}).
			Where("phone_hash = ? AND status = ?", p.PhoneHash, model.AppSubmitted).
			Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return code.ErrPhoneDuplicate
		}
		// 防重：同摘要已有 active 成员 → 4203
		var active int64
		if err := tx.Table("member_secure AS ms").
			Joins("JOIN member AS m ON m.id = ms.member_id AND m.deleted_at IS NULL AND m.status = ?", model.MemberActive).
			Where("ms.phone_hash = ?", p.PhoneHash).Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return code.ErrPhoneDuplicate
		}
		app := model.Application{
			InviteCodeID: ic.ID,
			Nickname:     p.Nickname,
			PhoneCipher:  p.PhoneCipher,
			PhoneHash:    p.PhoneHash,
			Status:       model.AppSubmitted,
		}
		if err := tx.Create(&app).Error; err != nil {
			return err
		}
		appID = app.ID
		return nil
	})
	return appID, err
}

// LatestByPhoneHash 最近一条申请（service.ApplicationLatestFinder）。
func (r *ApplicationRepo) LatestByPhoneHash(hash string) (*model.Application, error) {
	var a model.Application
	err := r.DB.Where("phone_hash = ?", hash).Order("id DESC").First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListByStatus 管理台列表（service.ApplicationLister）。
func (r *ApplicationRepo) ListByStatus(status string, offset, limit int) ([]model.Application, int64, error) {
	q := r.DB.Model(&model.Application{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Application
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

// FindByID 单条（service.ApplicationReviewer）。
func (r *ApplicationRepo) FindByID(id int64) (*model.Application, error) {
	var a model.Application
	if err := r.DB.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Approve 事务：建成员 + 隔离记录 → 回写申请 → 审计（service.ApplicationReviewer）。
func (r *ApplicationRepo) Approve(in service.ApproveInput) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(in.Member).Error; err != nil {
			return err
		}
		in.Secure.MemberID = in.Member.ID
		if err := tx.Create(in.Secure).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Application{}).Where("id = ?", in.TargetID).
			Updates(map[string]any{
				"status":      model.AppApproved,
				"reviewed_by": in.AdminID,
				"reviewed_at": time.Now(),
				"member_id":   in.Member.ID,
			}).Error; err != nil {
			return err
		}
		return appendAudit(tx, in.AdminID, in.Action, "application", in.TargetID, in.IP, in.UA)
	})
}

// Reject 拒绝（原因模板；service.ApplicationReviewer）。
func (r *ApplicationRepo) Reject(appID, adminID int64, reason, ip, ua string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.Application{}).
			Where("id = ? AND status = ?", appID, model.AppSubmitted).
			Updates(map[string]any{
				"status":        model.AppRejected,
				"reject_reason": reason,
				"reviewed_by":   adminID,
				"reviewed_at":   time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return code.ErrNotFound // 不存在或已结
		}
		return appendAudit(tx, adminID, "admin.reject", "application", appID, ip, ua)
	})
}
