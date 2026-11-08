package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/verifycode"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

// ---- Redis 实现（生产）与 Memory 实现（本地 dev 无 Redis 时）----
// 会话与频控数据全部在 Redis，不建表（架构设计 3.3）。

// 编译期接口断言。
var (
	_ verifycode.Store     = (*RedisVerifyStore)(nil)
	_ service.RefreshStore = (*RedisRefreshStore)(nil)
	_ service.RefreshStore = (*MemoryRefreshStore)(nil)
	_ service.FailCounter  = (*RedisFailCounter)(nil)
	_ service.FailCounter  = (*MemoryFailCounter)(nil)
)

// RedisVerifyStore 验证码存储（TTL 5 分钟 + 尝试上限）。
type RedisVerifyStore struct{ R *redis.Client }

func (s *RedisVerifyStore) Put(ctx context.Context, phoneHash, vc string) error {
	return s.R.Set(ctx, "sms:code:"+phoneHash, vc, verifycode.TTL).Err()
}

func (s *RedisVerifyStore) Verify(ctx context.Context, phoneHash, vc string) (bool, error) {
	stored, err := s.R.Get(ctx, "sms:code:"+phoneHash).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if stored != vc {
		n, err := s.R.Incr(ctx, "sms:att:"+phoneHash).Result()
		if err != nil {
			return false, err
		}
		if n == 1 {
			s.R.Expire(ctx, "sms:att:"+phoneHash, verifycode.TTL)
		}
		if n > verifycode.MaxAttempts {
			return false, verifycode.ErrTooManyAttempts
		}
		return false, nil
	}
	s.R.Del(ctx, "sms:code:"+phoneHash, "sms:att:"+phoneHash)
	return true, nil
}

// RedisRefreshStore 刷新令牌轮换状态（每成员当前 jti，TTL 30 天）。
type RedisRefreshStore struct{ R *redis.Client }

func (s *RedisRefreshStore) key(memberID int64) string {
	return "auth:refresh:" + strconv.FormatInt(memberID, 10)
}

func (s *RedisRefreshStore) Get(ctx context.Context, memberID int64) (string, error) {
	v, err := s.R.Get(ctx, s.key(memberID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return v, err
}

func (s *RedisRefreshStore) Set(ctx context.Context, memberID int64, jti string) error {
	return s.R.Set(ctx, s.key(memberID), jti, 30*24*time.Hour).Err()
}

func (s *RedisRefreshStore) Delete(ctx context.Context, memberID int64) error {
	return s.R.Del(ctx, s.key(memberID)).Err()
}

// MemoryRefreshStore 无 Redis 时的会话状态（dev / 单测）。
type MemoryRefreshStore struct {
	mu map[int64]string
}

func NewMemoryRefreshStore() *MemoryRefreshStore { return &MemoryRefreshStore{mu: map[int64]string{}} }

func (s *MemoryRefreshStore) Get(ctx context.Context, memberID int64) (string, error) {
	return s.mu[memberID], nil
}

func (s *MemoryRefreshStore) Set(ctx context.Context, memberID int64, jti string) error {
	s.mu[memberID] = jti
	return nil
}

func (s *MemoryRefreshStore) Delete(ctx context.Context, memberID int64) error {
	delete(s.mu, memberID)
	return nil
}

// RedisFailCounter 登录失败计数（同号 5 次锁 15 分钟）。
type RedisFailCounter struct{ R *redis.Client }

func (c *RedisFailCounter) Hit(ctx context.Context, key string, window time.Duration) (int64, error) {
	n, err := c.R.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		c.R.Expire(ctx, key, window)
	}
	return n, nil
}

func (c *RedisFailCounter) Count(ctx context.Context, key string) (int64, error) {
	n, err := c.R.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(n, 10, 64)
}

func (c *RedisFailCounter) Reset(ctx context.Context, key string) error {
	return c.R.Del(ctx, key).Err()
}

// MemoryFailCounter 无 Redis 时的失败计数。
type MemoryFailCounter struct {
	m      map[string]int64
	expiry map[string]time.Time
}

func NewMemoryFailCounter() *MemoryFailCounter {
	return &MemoryFailCounter{m: map[string]int64{}, expiry: map[string]time.Time{}}
}

func (c *MemoryFailCounter) Hit(ctx context.Context, key string, window time.Duration) (int64, error) {
	now := time.Now()
	if exp, ok := c.expiry[key]; ok && now.After(exp) {
		delete(c.m, key)
		delete(c.expiry, key)
	}
	c.m[key]++
	if c.m[key] == 1 {
		c.expiry[key] = now.Add(window)
	}
	return c.m[key], nil
}

func (c *MemoryFailCounter) Count(ctx context.Context, key string) (int64, error) {
	if exp, ok := c.expiry[key]; ok && time.Now().After(exp) {
		return 0, nil
	}
	return c.m[key], nil
}

func (c *MemoryFailCounter) Reset(ctx context.Context, key string) error {
	delete(c.m, key)
	delete(c.expiry, key)
	return nil
}
