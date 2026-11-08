package repository

import (
	"gorm.io/gorm"
)

// Counters 我的页三项统计（service.Counters）。
type Counters struct{ DB *gorm.DB }

func (c *Counters) CountEnrollments(memberID int64) (int64, error) {
	var n int64
	err := c.DB.Table("enrollment").Where("member_id = ? AND status = 'enrolled'", memberID).Count(&n).Error
	return n, err
}

func (c *Counters) CountPosts(memberID int64) (int64, error) {
	var n int64
	err := c.DB.Table("post").Where("author_id = ? AND status = 'published'", memberID).Count(&n).Error
	return n, err
}

func (c *Counters) CountWorks(memberID int64) (int64, error) {
	// 孩子的作品计数按本人报名活动聚合（M4 接 audience 精确口径，先按报名数兜底）
	var n int64
	err := c.DB.Table("enrollment").Where("member_id = ? AND status = 'enrolled'", memberID).Count(&n).Error
	return n, err
}
