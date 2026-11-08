// Package idempot 写操作幂等（架构设计 3.4 模式 1）。
//
// 铁律：**幂等检查必须放在频控之前**——否则重试请求先撞频控，永远无法幂等通过
// （参考架构真实踩坑）。调用顺序固定：Acquire →（首次）执行业务 → Save 首次结果；
// 重复请求 Load 直接返回首次结果。
package idempot

import (
	"context"
	"sync"
	"time"
)

// Store 幂等存储：key 为 "idem:{token}"，TTL 24 小时（细化设计 6）。
type Store interface {
	// Acquire 抢占：首次返回 true；重复返回 false（此时应 Load 首次结果）。
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)
	// Save 保存首次执行结果（重复请求的返回值）。
	Save(ctx context.Context, key string, payload []byte, ttl time.Duration) error
	// Load 取首次结果；不存在返回 (nil, false, nil)。
	Load(ctx context.Context, key string) ([]byte, bool, error)
	// Release 释放槽位：首次请求在业务上失败（频控/校验等）时调用，
	// 让客户端的重试可以重新执行，而不是永远撞在占位上。
	Release(ctx context.Context, key string) error
}

// MemoryStore 内存实现（单测 / 本地 dev；生产用 RedisStore）。
type MemoryStore struct {
	mu  sync.Mutex
	m   map[string][]byte
	exp map[string]time.Time
}

func NewMemory() *MemoryStore {
	return &MemoryStore{m: map[string][]byte{}, exp: map[string]time.Time{}}
}

func (s *MemoryStore) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if exp, ok := s.exp[key]; ok && time.Now().After(exp) {
		delete(s.m, key)
		delete(s.exp, key)
	}
	if _, ok := s.m[key]; ok {
		return false, nil
	}
	s.m[key] = nil // 占位：结果尚未 Save
	s.exp[key] = time.Now().Add(ttl)
	return true, nil
}

func (s *MemoryStore) Save(ctx context.Context, key string, payload []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = payload
	return nil
}

func (s *MemoryStore) Load(ctx context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[key]
	if !ok || v == nil {
		return nil, false, nil
	}
	return v, true, nil
}

func (s *MemoryStore) Release(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
	delete(s.exp, key)
	return nil
}
