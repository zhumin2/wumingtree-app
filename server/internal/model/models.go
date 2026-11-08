// Package model GORM 模型，与 migrations/000001_init.up.sql 一一对应。
// 表结构唯一来源是迁移 SQL；本包只做映射，禁止 AutoMigrate（架构设计 3.7）。
package model

import (
	"time"

	"gorm.io/gorm"
)

// 角色与状态常量（DDL 注释同源）。
const (
	RoleMember  = "member"
	RoleTeacher = "teacher"
	RoleAdmin   = "admin"

	MemberActive   = "active"
	MemberDisabled = "disabled"

	AppSubmitted = "submitted"
	AppApproved  = "approved"
	AppRejected  = "rejected"

	InviteActive    = "active"
	InviteDisabled  = "disabled"
	InviteExhausted = "exhausted"
)

// Member 成员主表：不落任何手机号字段（红线，member_secure 隔离存储）。
type Member struct {
	ID           int64          `gorm:"primaryKey;autoIncrement"`
	Nickname     string         `gorm:"column:nickname;size:8;not null"`
	Role         string         `gorm:"column:role;size:16;not null;default:member"`
	Status       string         `gorm:"column:status;size:16;not null;default:active"`
	DefaultAnon  bool           `gorm:"column:default_anon;not null;default:true"`
	DND          bool           `gorm:"column:dnd;not null;default:true"`
	NotifyNotice bool           `gorm:"column:notify_notice;not null;default:true"`
	NotifyHollow bool           `gorm:"column:notify_hollow;not null;default:true"`
	JoinedAt     time.Time      `gorm:"column:joined_at;not null"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at"` // 软删（成员不物理删除）
	CreatedAt    time.Time      `gorm:"column:created_at;not null"`
}

func (Member) TableName() string { return "member" }

// MemberSecure 手机号隔离存储：密文（base64 于 VARBINARY）+ HMAC 摘要。
type MemberSecure struct {
	MemberID    int64     `gorm:"primaryKey;column:member_id"`
	PhoneCipher string    `gorm:"column:phone_cipher;not null"`
	PhoneHash   string    `gorm:"column:phone_hash;size:64;not null;uniqueIndex"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null"`
}

func (MemberSecure) TableName() string { return "member_secure" }

// InviteCode 邀请码。
type InviteCode struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	Code       string    `gorm:"column:code;size:32;not null;uniqueIndex"`
	Status     string    `gorm:"column:status;size:16;not null;default:active"`
	ValidFrom  time.Time `gorm:"column:valid_from;not null"`
	ValidUntil time.Time `gorm:"column:valid_until;not null"`
	MaxUses    int       `gorm:"column:max_uses;not null;default:1"`
	UsedCount  int       `gorm:"column:used_count;not null;default:0"`
	CreatedBy  int64     `gorm:"column:created_by;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;not null"`
}

func (InviteCode) TableName() string { return "invite_code" }

// Application 加入申请。
type Application struct {
	ID           int64      `gorm:"primaryKey;autoIncrement"`
	InviteCodeID int64      `gorm:"column:invite_code_id;not null"`
	Nickname     string     `gorm:"column:nickname;size:8;not null"`
	PhoneCipher  string     `gorm:"column:phone_cipher;not null"`
	PhoneHash    string     `gorm:"column:phone_hash;size:64;not null"`
	Status       string     `gorm:"column:status;size:16;not null;default:submitted"`
	RejectReason *string    `gorm:"column:reject_reason;size:200"`
	ReviewedBy   *int64     `gorm:"column:reviewed_by"`
	ReviewedAt   *time.Time `gorm:"column:reviewed_at"`
	MemberID     *int64     `gorm:"column:member_id"`
	CreatedAt    time.Time  `gorm:"column:created_at;not null"`
}

func (Application) TableName() string { return "application" }

// AuditLog 审计（手机号明文查看、管理操作、导出等；架构设计 5.4）。
type AuditLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	ActorID    *int64    `gorm:"column:actor_id"`
	Action     string    `gorm:"column:action;size:50;not null"`
	TargetType *string   `gorm:"column:target_type;size:16"`
	TargetID   *int64    `gorm:"column:target_id"`
	IP         *string   `gorm:"column:ip;size:45"`
	UserAgent  *string   `gorm:"column:user_agent;size:255"`
	OccurredAt time.Time `gorm:"column:occurred_at;not null"`
}

func (AuditLog) TableName() string { return "audit_log" }
