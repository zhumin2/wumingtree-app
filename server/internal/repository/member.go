// Package repository GORM 数据访问：一个业务域一个 Repo（架构设计 3.1）。
// 直接实现 service 层定义的接口（依赖倒置）；事务内才能保证一致性的操作整体封装在此。
package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

// MemberRepo 成员与手机号隔离存储。
type MemberRepo struct{ DB *gorm.DB }

// CreateWithSecure 同一事务创建成员与隔离记录。
func (r *MemberRepo) CreateWithSecure(m *model.Member, s *model.MemberSecure) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		s.MemberID = m.ID
		return tx.Create(s).Error
	})
}

// FindSecureByPhoneHash 按摘要取成员与隔离记录（service.MemberFinder）。
// 无记录返回 (nil, nil, nil)，由 service 决定下一步。
func (r *MemberRepo) FindSecureByPhoneHash(hash string) (*model.Member, *model.MemberSecure, error) {
	var s model.MemberSecure
	err := r.DB.Where("phone_hash = ?", hash).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var m model.Member
	if err := r.DB.First(&m, s.MemberID).Error; err != nil {
		return nil, nil, err
	}
	return &m, &s, nil
}

// FindByID 按 ID 取成员（service.MemberByID / service.MeReader）。
func (r *MemberRepo) FindByID(id int64) (*model.Member, error) {
	var m model.Member
	if err := r.DB.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// SecureCipherByMemberID 取本人手机号密文（我的页掩码展示用）。
func (r *MemberRepo) SecureCipherByMemberID(memberID int64) (string, error) {
	var s model.MemberSecure
	if err := r.DB.Select("phone_cipher").Where("member_id = ?", memberID).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return s.PhoneCipher, nil
}

// UpdateProfile 更新本人资料（service.ProfileUpdater）。
func (r *MemberRepo) UpdateProfile(id int64, p service.MemberPatch) error {
	updates := map[string]any{}
	if p.Nickname != nil {
		updates["nickname"] = *p.Nickname
	}
	if p.DefaultAnon != nil {
		updates["default_anon"] = *p.DefaultAnon
	}
	if p.DND != nil {
		updates["dnd"] = *p.DND
	}
	if p.NotifyNotice != nil {
		updates["notify_notice"] = *p.NotifyNotice
	}
	if p.NotifyHollow != nil {
		updates["notify_hollow"] = *p.NotifyHollow
	}
	if len(updates) == 0 {
		return nil
	}
	return r.DB.Model(&model.Member{}).Where("id = ?", id).Updates(updates).Error
}

// AuditRepo 审计日志（架构设计 5.4）。
type AuditRepo struct{ DB *gorm.DB }

func (r *AuditRepo) Append(actorID int64, action, targetType string, targetID int64, ip, ua string) error {
	return appendAudit(r.DB, actorID, action, targetType, targetID, ip, ua)
}

func appendAudit(tx *gorm.DB, actorID int64, action, targetType string, targetID int64, ip, ua string) error {
	e := model.AuditLog{
		ActorID:    &actorID,
		Action:     action,
		TargetType: &targetType,
		TargetID:   &targetID,
		OccurredAt: time.Now(),
	}
	if ip != "" {
		e.IP = &ip
	}
	if ua != "" {
		e.UserAgent = &ua
	}
	return tx.Create(&e).Error
}

// ensure code 包在重构后仍被引用（防误删 import）。
var _ = code.CodeOK
