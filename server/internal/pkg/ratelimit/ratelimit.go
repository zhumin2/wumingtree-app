// Package ratelimit 频控（架构设计 3.4 模式 2）。
// Redis INCR key:窗口 + 首次 EXPIRE；Memory 实现用于单测与本地 dev。
// 注意：调用方必须保证「幂等检查先于频控」（M2 接入 client_token 时执行，架构设计 3.4 铁律）。
package ratelimit

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// Limiter 固定窗口限流。key 已由调用方带上业务前缀与主体（如 sms:freq:{phoneHash}）。
type Limiter interface {
	// Allow 窗口内第 n 次调用：n <= limit 返回 true 并计数，否则 false（不计超额）。
	Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error)
}

// MemoryLimiter 内存实现（单测 / 本地 dev）。
type MemoryLimiter struct {
	mu      sync.Mutex
	counts  map[string]int64
	expires map[string]time.Time
}

func NewMemory() *MemoryLimiter {
	return &MemoryLimiter{counts: map[string]int64{}, expires: map[string]time.Time{}}
}

func (l *MemoryLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if exp, ok := l.expires[key]; ok && now.After(exp) {
		delete(l.counts, key)
		delete(l.expires, key)
	}
	l.counts[key]++
	if l.counts[key] == 1 {
		l.expires[key] = now.Add(window)
	}
	return l.counts[key] <= limit, nil
}

// RedisLimiter 生产实现：INCR + 首次 EXPIRE。
type RedisLimiter struct {
	R RedisCmd
}

// RedisCmd 最小接口，便于单测注入（生产由 go-redis 适配）。
type RedisCmd interface {
	Incr(ctx context.Context, key string) (int64, error)
	ExpireNX(ctx context.Context, key string, window time.Duration) error
}

func (l *RedisLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	n, err := l.R.Incr(ctx, key)
	if err != nil {
		return false, err
	}
	if n == 1 {
		if err := l.R.ExpireNX(ctx, key, window); err != nil {
			return false, err
		}
	}
	return n <= limit, nil
}

// FormatCount 便于日志输出计数（避免 strconv 散落）。
func FormatCount(n int64) string { return strconv.FormatInt(n, 10) }
