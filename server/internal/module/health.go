// Package module HTTP handler 层：只做参数绑定、调 service、按错误映射信封（架构设计 3.1）。
// 本文件是分层示例的最小实现；业务模块自 M1 起按此模式挂载。
package module

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/httpx"
)

// Health 组件健康态。
type Health struct {
	Status string `json:"status"`
	DB     bool   `json:"db"`
	Redis  bool   `json:"redis"`
}

// NewHealth 返回 healthz handler。db / rdb 允许为 nil（未配置时报告 false 而非崩溃，
// 便于本地与 CI 无依赖启动）。
func NewHealth(db *gorm.DB, pingRedis func(ctx context.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := Health{Status: "ok"}
		if db != nil {
			if sqlDB, err := db.DB(); err == nil {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
				h.DB = sqlDB.PingContext(ctx) == nil
				cancel()
			}
		}
		if pingRedis != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			h.Redis = pingRedis(ctx) == nil
			cancel()
		}
		httpx.OK(c, h)
	}
}
