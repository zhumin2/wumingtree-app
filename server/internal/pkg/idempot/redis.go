package idempot

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore 生产实现（SETNX + GET/SET + DEL）。
type RedisStore struct{ R *redis.Client }

func (s *RedisStore) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	ok, err := s.R.SetNX(ctx, key, "__pending__", ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (s *RedisStore) Save(ctx context.Context, key string, payload []byte, ttl time.Duration) error {
	return s.R.Set(ctx, key, payload, ttl).Err()
}

func (s *RedisStore) Load(ctx context.Context, key string) ([]byte, bool, error) {
	v, err := s.R.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if v == "__pending__" {
		return nil, false, nil // 首次请求仍在执行
	}
	return []byte(v), true, nil
}

func (s *RedisStore) Release(ctx context.Context, key string) error {
	return s.R.Del(ctx, key).Err()
}
