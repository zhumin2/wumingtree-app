package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/cursor"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

// 行结构（匿名嵌入：gorm 按 post.* 列映射；作者昵称随行返回，
// 匿名显示名由 service 解析——红线：匿名内容 API 不回传作者字段）。
type PostRow struct {
	model.Post     `gorm:"embedded"` // 显式内嵌：否则 gorm 按 has-one 关联解析要求外键
	AuthorNickname string            `gorm:"column:author_nickname"`
}

type FloorRow struct {
	model.Floor    `gorm:"embedded"` // 同上
	AuthorNickname string            `gorm:"column:author_nickname"`
}

// ErrNotFound 举报对象不存在或已结（repository 层别名）。
var ErrNotFound = gorm.ErrRecordNotFound

// PostRepo 树洞帖。
type PostRepo struct{ DB *gorm.DB }

func postRowQuery(db *gorm.DB) *gorm.DB {
	return db.Table("post").
		Select("post.*, member.nickname AS author_nickname").
		Joins("LEFT JOIN member ON member.id = post.author_id")
}

func (r *PostRepo) Create(p *model.Post) error { return r.DB.Create(p).Error }

// FindByID 满足 service.PostStore。不存在返回 (nil, nil)。
func (r *PostRepo) FindByID(id int64) (*service.PostRow, error) {
	var row service.PostRow
	err := postRowQuery(r.DB).Where("post.id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// List 心情筛选 + 时间倒序游标分页（service.PostStore）。
func (r *PostRepo) List(mood string, page service.Page) ([]service.PostRow, string, error) {
	q := postRowQuery(r.DB).Where("post.status = ?", model.PostPublished)
	if mood != "" {
		q = q.Where("post.mood = ?", mood)
	}
	if at, id, has, err := cursor.Parse(page.Cursor); err != nil {
		return nil, "", err
	} else if has {
		q = q.Where("(post.published_at < ? OR (post.published_at = ? AND post.id < ?))", at, at, id)
	}
	var rows []service.PostRow
	if err := q.Order("post.published_at DESC, post.id DESC").Limit(page.Limit + 1).Scan(&rows).Error; err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > page.Limit {
		last := rows[page.Limit-1]
		next = cursor.Key(last.Post.PublishedAt, last.Post.ID)
		rows = rows[:page.Limit]
	}
	return rows, next, nil
}

// ListByAuthor 我的树洞（service.PostStore）。
func (r *PostRepo) ListByAuthor(authorID int64, page service.Page) ([]service.PostRow, string, error) {
	q := postRowQuery(r.DB).
		Where("post.status = ? AND post.author_id = ?", model.PostPublished, authorID)
	if at, id, has, err := cursor.Parse(page.Cursor); err != nil {
		return nil, "", err
	} else if has {
		q = q.Where("(post.published_at < ? OR (post.published_at = ? AND post.id < ?))", at, at, id)
	}
	var rows []service.PostRow
	if err := q.Order("post.published_at DESC, post.id DESC").Limit(page.Limit + 1).Scan(&rows).Error; err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > page.Limit {
		last := rows[page.Limit-1]
		next = cursor.Key(last.Post.PublishedAt, last.Post.ID)
		rows = rows[:page.Limit]
	}
	return rows, next, nil
}

// UpdateStatus 软删（作者删除 / 举报处置）。
func (r *PostRepo) UpdateStatus(id int64, status, removedReason string) error {
	return r.DB.Model(&model.Post{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "removed_reason": removedReason}).Error
}

// FloorRepo 楼层。
type FloorRepo struct{ DB *gorm.DB }

func floorRowQuery(db *gorm.DB) *gorm.DB {
	return db.Table("floor").
		Select("floor.*, member.nickname AS author_nickname").
		Joins("LEFT JOIN member ON member.id = floor.author_id")
}

func (r *FloorRepo) Create(f *model.Floor) error { return r.DB.Create(f).Error }

// FindByID 满足 service.FloorStore。
func (r *FloorRepo) FindByID(id int64) (*model.Floor, error) {
	var f model.Floor
	if err := r.DB.First(&f, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

// CountPublished 满足 service.FloorStore。
func (r *FloorRepo) CountPublished(postID int64) (int64, error) {
	var n int64
	err := r.DB.Model(&model.Floor{}).
		Where("post_id = ? AND status = ?", postID, model.FloorPublished).Count(&n).Error
	return n, err
}

// CountPublishedByPosts 批量回应数（service.FloorStore）。
func (r *FloorRepo) CountPublishedByPosts(postIDs []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(postIDs) == 0 {
		return out, nil
	}
	type agg struct {
		PostID int64
		N      int64
	}
	var rows []agg
	err := r.DB.Model(&model.Floor{}).
		Select("post_id, COUNT(*) AS n").
		Where("post_id IN ? AND status = ?", postIDs, model.FloorPublished).
		Group("post_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.PostID] = x.N
	}
	return out, nil
}

// ListByPost 楼层按时间从上到下（service.FloorStore）。
func (r *FloorRepo) ListByPost(postID int64, page service.Page) ([]service.FloorRow, string, error) {
	q := floorRowQuery(r.DB).
		Where("floor.post_id = ? AND floor.status = ?", postID, model.FloorPublished)
	if at, id, has, err := cursor.Parse(page.Cursor); err != nil {
		return nil, "", err
	} else if has {
		q = q.Where("(floor.created_at < ? OR (floor.created_at = ? AND floor.id < ?))", at, at, id)
	}
	var rows []service.FloorRow
	if err := q.Order("floor.created_at ASC, floor.id ASC").Limit(page.Limit + 1).Scan(&rows).Error; err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > page.Limit {
		last := rows[page.Limit-1]
		next = cursor.Key(last.Floor.CreatedAt, last.Floor.ID)
		rows = rows[:page.Limit]
	}
	return rows, next, nil
}

// ReactionRepo 回应（toggle：有则删、无则插；并发由唯一约束兜底）。
type ReactionRepo struct{ DB *gorm.DB }

func (r *ReactionRepo) Toggle(memberID int64, targetType string, targetID int64, rtype string) (bool, error) {
	active := false
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("member_id = ? AND target_type = ? AND target_id = ? AND type = ?",
			memberID, targetType, targetID, rtype).Delete(&model.Reaction{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			active = false
			return nil
		}
		rec := model.Reaction{MemberID: memberID, TargetType: targetType, TargetID: targetID, Type: rtype}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		active = true
		return nil
	})
	return active, err
}

func (r *ReactionRepo) Count(targetType string, targetID int64, rtype string) (int64, error) {
	var n int64
	err := r.DB.Model(&model.Reaction{}).
		Where("target_type = ? AND target_id = ? AND type = ?", targetType, targetID, rtype).Count(&n).Error
	return n, err
}

// Mine 满足 service.ReactionStore。
func (r *ReactionRepo) Mine(memberID int64, targetType string, targetID int64) ([]string, error) {
	var rows []model.Reaction
	err := r.DB.Where("member_id = ? AND target_type = ? AND target_id = ?",
		memberID, targetType, targetID).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, x := range rows {
		out = append(out, x.Type)
	}
	return out, nil
}

// MineByTargets 批量（列表页用）：目标 ID -> 回应类型列表。
func (r *ReactionRepo) MineByTargets(memberID int64, targetType string, ids []int64) (map[int64][]string, error) {
	out := map[int64][]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []model.Reaction
	err := r.DB.Where("member_id = ? AND target_type = ? AND target_id IN ?",
		memberID, targetType, ids).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.TargetID] = append(out[x.TargetID], x.Type)
	}
	return out, nil
}

// CountByTargets 批量计数：目标 ID -> {type: count}。
func (r *ReactionRepo) CountByTargets(targetType string, ids []int64) (map[int64]map[string]int64, error) {
	out := map[int64]map[string]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	type agg struct {
		TargetID int64
		Type     string
		N        int64
	}
	var rows []agg
	err := r.DB.Model(&model.Reaction{}).
		Select("target_id, type, COUNT(*) AS n").
		Where("target_type = ? AND target_id IN ?", targetType, ids).
		Group("target_id, type").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		if out[x.TargetID] == nil {
			out[x.TargetID] = map[string]int64{}
		}
		out[x.TargetID][x.Type] = x.N
	}
	return out, nil
}

// ReportRepo 举报。
type ReportRepo struct{ DB *gorm.DB }

func (r *ReportRepo) Create(rep *model.Report) error { return r.DB.Create(rep).Error }

func (r *ReportRepo) ListByStatus(status string, offset, limit int) ([]model.Report, int64, error) {
	q := r.DB.Model(&model.Report{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Report
	err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error // 队列按受理顺序
	return list, total, err
}

// Handle 处理举报：remove_content 时事务内联动下架目标内容。
func (r *ReportRepo) Handle(reportID, adminID int64, action, note string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var rep model.Report
		if err := tx.First(&rep, reportID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if rep.Status != model.ReportOpen {
			return ErrNotFound // 已结
		}
		now := time.Now()
		if err := tx.Model(&model.Report{}).Where("id = ?", reportID).
			Updates(map[string]any{
				"status":     model.ReportHandled,
				"handled_by": adminID,
				"handled_at": now,
				"action":     action,
				"note":       note,
			}).Error; err != nil {
			return err
		}
		if action == "remove_content" {
			switch rep.TargetType {
			case "post":
				if err := tx.Model(&model.Post{}).Where("id = ?", rep.TargetID).
					Updates(map[string]any{"status": model.PostRemoved, "removed_reason": "by_report"}).Error; err != nil {
					return err
				}
			case "floor":
				if err := tx.Model(&model.Floor{}).Where("id = ?", rep.TargetID).
					Updates(map[string]any{"status": model.FloorRemoved}).Error; err != nil {
					return err
				}
			}
		}
		return appendAudit(tx, adminID, "admin.handle_report", rep.TargetType, rep.TargetID, "", "")
	})
}

// ---- 红点存储（仅置位/清除/读取布尔，无聚合计数——红线）----

// MemoryBadgeStore 内存实现（单测 / 本地 dev；生产用 RedisBadgeStore）。
type MemoryBadgeStore struct {
	mu map[int64]map[string]bool
}

func NewMemoryBadgeStore() *MemoryBadgeStore {
	return &MemoryBadgeStore{mu: map[int64]map[string]bool{}}
}

func (s *MemoryBadgeStore) Mark(ctx context.Context, memberID int64, kind string) error {
	if s.mu[memberID] == nil {
		s.mu[memberID] = map[string]bool{}
	}
	s.mu[memberID][kind] = true
	return nil
}

func (s *MemoryBadgeStore) Clear(ctx context.Context, memberID int64, kind string) error {
	if s.mu[memberID] != nil {
		delete(s.mu[memberID], kind)
	}
	return nil
}

func (s *MemoryBadgeStore) Read(ctx context.Context, memberID int64, kind string) (bool, error) {
	return s.mu[memberID][kind], nil
}

// RedisBadgeStore 生产实现：badges:{memberID} 哈希。
type RedisBadgeStore struct{ R *redis.Client }

func (s *RedisBadgeStore) Mark(ctx context.Context, memberID int64, kind string) error {
	return s.R.HSet(ctx, badgeKey(memberID), kind, 1).Err()
}

func (s *RedisBadgeStore) Clear(ctx context.Context, memberID int64, kind string) error {
	return s.R.HDel(ctx, badgeKey(memberID), kind).Err()
}

func (s *RedisBadgeStore) Read(ctx context.Context, memberID int64, kind string) (bool, error) {
	v, err := s.R.HGet(ctx, badgeKey(memberID), kind).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == "1", nil
}

func badgeKey(memberID int64) string { return "badges:" + strconv.FormatInt(memberID, 10) }
