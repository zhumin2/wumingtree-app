// Package router 路由装配：中间件 → 健康检查 → 业务模块挂载（依赖接线）。
package router

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/zhumin2/wumingtree-app/server/internal/config"
	"github.com/zhumin2/wumingtree-app/server/internal/middleware"
	"github.com/zhumin2/wumingtree-app/server/internal/module"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/audit"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/crypto"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/httpx"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/idempot"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/ratelimit"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/sms"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/tc3"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/verifycode"
	"github.com/zhumin2/wumingtree-app/server/internal/repository"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

var errRedisNotConfigured = errors.New("redis not configured")

// New 装配路由。db / rdb 允许为 nil（本地与 CI 无依赖启动，业务路由不挂载）。
func New(cfg config.Config, db *gorm.DB, rdb *redis.Client) *gin.Engine {
	if cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID())

	// 客户端解析器只认信封：未匹配路由也走信封（5001 口径）。
	r.NoRoute(func(c *gin.Context) {
		httpx.FailCode(c, code.CodeNotFound, code.ErrNotFound.Message)
	})

	pingRedis := func(ctx context.Context) error { return errRedisNotConfigured }
	if rdb != nil {
		pingRedis = func(ctx context.Context) error { return rdb.Ping(ctx).Err() }
	}

	api := r.Group("/api/v1")
	api.GET("/healthz", module.NewHealth(db, pingRedis))

	// 用户端 Web（参考架构 5.2）：Flutter Web 产物同源托管 + no-cache 铁律。
	// hash 路由策略下浏览器只请求 /static/web/ 本身，无需 SPA fallback。
	if cfg.WebDir != "" {
		if _, err := os.Stat(cfg.WebDir); err == nil {
			noCache := func(c *gin.Context) {
				c.Header("Cache-Control", "no-cache, must-revalidate")
				c.Next()
			}
			web := r.Group("/static/web", noCache)
			web.Static("/", cfg.WebDir)
			slog.Info("用户端 Web 已托管", "dir", cfg.WebDir)
		} else {
			slog.Warn("WEB_DIR 不存在，用户端 Web 未托管", "dir", cfg.WebDir)
		}
	}

	if db == nil {
		slog.Warn("未连接数据库，业务路由未挂载（仅 healthz 可用）")
		return r
	}

	// ---- 手机号加解密器（FILE_KEY 32 字节）----
	key, err := hex.DecodeString(cfg.FileKey)
	if err != nil || len(key) != 32 {
		slog.Error("FILE_KEY 非法，业务路由未挂载", "err", err)
		return r
	}
	phoneCrypto, err := crypto.NewPhoneCrypto(key)
	if err != nil {
		slog.Error("手机号加解密器构造失败", "err", err)
		return r
	}

	// ---- 仓库 ----
	memberRepo := &repository.MemberRepo{DB: db}
	appRepo := &repository.ApplicationRepo{DB: db}
	counters := &repository.Counters{DB: db}
	postRepo := &repository.PostRepo{DB: db}
	floorRepo := &repository.FloorRepo{DB: db}
	reactionRepo := &repository.ReactionRepo{DB: db}
	reportRepo := &repository.ReportRepo{DB: db}

	// ---- Redis 会话件（无 Redis 时退化为内存实现，仅 dev）----
	var vstore verifycode.Store = verifycode.NewMemory()
	var lim ratelimit.Limiter = ratelimit.NewMemory()
	var fails service.FailCounter = repository.NewMemoryFailCounter()
	var refresh service.RefreshStore = repository.NewMemoryRefreshStore()
	var badges service.BadgeStore = repository.NewMemoryBadgeStore()
	if rdb != nil {
		vstore = &repository.RedisVerifyStore{R: rdb}
		lim = &ratelimit.RedisLimiter{R: &redisCmdAdapter{R: rdb}}
		fails = &repository.RedisFailCounter{R: rdb}
		refresh = &repository.RedisRefreshStore{R: rdb}
		badges = &repository.RedisBadgeStore{R: rdb}
	} else {
		slog.Warn("未配置 Redis，会话与频控使用内存实现（仅限 dev）")
	}

	// ---- 服务 ----
	tencentCred := tc3.Credential{SecretID: cfg.TencentSecretID, SecretKey: cfg.TencentSecretKey}
	authSvc := service.NewAuthService(
		memberRepo, memberRepo, appRepo, vstore,
		sms.New(cfg.SMSType == "mock", sms.Options{
			TC:         tc3.New(tencentCred, "sms.tencentcloudapi.com", "sms", cfg.TencentRegion),
			SdkAppID:   cfg.SmsSdkAppID,
			SignName:   cfg.SmsSignName,
			TemplateID: cfg.SmsTemplateID,
		}),
		lim, fails, refresh, phoneCrypto,
		cfg.SMSType == "mock", cfg.JWTSecret,
	)
	appSvc := service.NewApplicationService(appRepo, appRepo, appRepo, phoneCrypto)
	auditSvc := service.NewAuditWriter(memberRepo, phoneCrypto, &repository.AuditRepo{DB: db})
	meSvc := service.NewMemberService(memberRepo, memberRepo, memberRepo, counters, badges, phoneCrypto)
	auditor := audit.New(cfg.AuditMode, audit.Options{
		TC: tc3.New(tencentCred, "tms.tencentcloudapi.com", "tms", cfg.TencentRegion),
	})
	hollowSvc := service.NewHollowService(
		postRepo, floorRepo, reactionRepo, memberRepo,
		&repository.AuditRepo{DB: db}, auditor,
		lim, idempot.NewMemory(), badges,
	)
	reportSvc := service.NewReportService(reportRepo, reportRepo, reportRepo, lim)

	// ---- 路由 ----
	module.RegisterAuth(api, authSvc)
	module.RegisterApplications(api, appSvc, auditSvc, middleware.Auth(cfg.JWTSecret), middleware.RequireAdmin())
	module.RegisterHollow(api, hollowSvc, reportSvc, middleware.Auth(cfg.JWTSecret), middleware.RequireAdmin())

	me := api.Group("", middleware.Auth(cfg.JWTSecret))
	module.RegisterMember(me, meSvc)

	// M3 挂载点：notices / folders / files；
	// M4：activities / enrollments / archive；M5：conversations。
	return r
}

// redisCmdAdapter 适配 go-redis 到 ratelimit.RedisCmd 最小接口。
type redisCmdAdapter struct{ R *redis.Client }

func (a *redisCmdAdapter) Incr(ctx context.Context, key string) (int64, error) {
	return a.R.Incr(ctx, key).Result()
}

func (a *redisCmdAdapter) ExpireNX(ctx context.Context, key string, window time.Duration) error {
	return a.R.ExpireNX(ctx, key, window).Err()
}
