package model

import "time"

// 树洞内容状态常量（DDL 注释同源）。
const (
	PostPublished = "published"
	PostRemoved   = "removed"

	FloorPublished = "published"
	FloorRemoved   = "removed"

	ReviewClean   = "clean"
	ReviewPending = "pending"

	ReportOpen    = "open"
	ReportHandled = "handled"
	ReportDismiss = "dismissed"
)

// Post 树洞帖。author_id 始终保留（删除权与治理）；匿名时 API 不回传——红线。
type Post struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	AuthorID      int64     `gorm:"column:author_id;not null"`
	Mood          string    `gorm:"column:mood;size:8;not null"`
	Title         string    `gorm:"column:title;size:50;not null"`
	Body          *string   `gorm:"column:body"`
	IsAnonymous   bool      `gorm:"column:is_anonymous;not null;default:true"`
	Status        string    `gorm:"column:status;size:16;not null;default:published"`
	RemovedReason *string   `gorm:"column:removed_reason;size:200"`
	ReviewStatus  string    `gorm:"column:review_status;size:16;not null;default:clean"`
	PublishedAt   time.Time `gorm:"column:published_at;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;not null"`
}

func (Post) TableName() string { return "post" }

// Floor 楼层（楼中楼仅一层：depth 与 parent_floor_id 配对由 CHECK 强制）。
type Floor struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	PostID        int64     `gorm:"column:post_id;not null"`
	ParentFloorID *int64    `gorm:"column:parent_floor_id"`
	Depth         int8      `gorm:"column:depth;not null;default:1"`
	AuthorID      int64     `gorm:"column:author_id;not null"`
	Content       string    `gorm:"column:content;size:500;not null"`
	IsAnonymous   bool      `gorm:"column:is_anonymous;not null;default:true"`
	Status        string    `gorm:"column:status;size:16;not null;default:published"`
	ReviewStatus  string    `gorm:"column:review_status;size:16;not null;default:clean"`
	CreatedAt     time.Time `gorm:"column:created_at;not null"`
}

func (Floor) TableName() string { return "floor" }

// Reaction 回应（toggle 由唯一约束实现；无负向类型——红线）。
type Reaction struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	MemberID   int64     `gorm:"column:member_id;not null"`
	TargetType string    `gorm:"column:target_type;size:8;not null"`
	TargetID   int64     `gorm:"column:target_id;not null"`
	Type       string    `gorm:"column:type;size:8;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;not null"`
}

func (Reaction) TableName() string { return "reaction" }

// Report 举报（目标多态；reporter_id 任何接口不回传——红线）。
type Report struct {
	ID         int64      `gorm:"primaryKey;autoIncrement"`
	TargetType string     `gorm:"column:target_type;size:16;not null"`
	TargetID   int64      `gorm:"column:target_id;not null"`
	Reason     string     `gorm:"column:reason;size:8;not null"`
	Detail     *string    `gorm:"column:detail;size:200"`
	ReporterID int64      `gorm:"column:reporter_id;not null"`
	Status     string     `gorm:"column:status;size:16;not null;default:open"`
	HandledBy  *int64     `gorm:"column:handled_by"`
	HandledAt  *time.Time `gorm:"column:handled_at"`
	Action     *string    `gorm:"column:action;size:50"`
	Note       *string    `gorm:"column:note;size:200"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
}

func (Report) TableName() string { return "report" }
