// 无名树 App API 服务入口：配置 → MySQL/Redis → 路由 → 启动。
// 后台任务（Goroutine + Ticker + Redis 单实例锁）自 M4 接入（架构设计 3.8）。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/zhumin2/wumingtree-app/server/internal/config"
	"github.com/zhumin2/wumingtree-app/server/internal/router"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("配置加载失败", "err", err)
		os.Exit(1)
	}

	var db *gorm.DB
	if cfg.DBDSN != "" {
		if d, err := gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{}); err != nil {
			slog.Warn("MySQL 连接失败（healthz 将报告 db=false）", "err", err)
		} else {
			db = d
			slog.Info("MySQL 已连接")
		}
	} else {
		slog.Warn("未配置 DB_DSN，跳过 MySQL 连接（healthz 将报告 db=false）")
	}

	var rdb *redis.Client
	if cfg.RedisAddr != "" {
		rdb = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
		slog.Info("Redis 客户端已初始化", "addr", cfg.RedisAddr)
	} else {
		slog.Warn("未配置 REDIS_ADDR，跳过 Redis")
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router.New(cfg, db, rdb),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("API 启动", "addr", cfg.Addr, "env", cfg.Env, "sms", cfg.SMSType, "audit", cfg.AuditMode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP 服务异常退出", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("收到退出信号，优雅停机")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("停机超时", "err", err)
	}
}
