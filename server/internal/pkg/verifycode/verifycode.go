// Package verifycode 短信验证码存储（架构设计 3.3）。
// Redis TTL 5 分钟、错误尝试上限 5 次（不建表，细化设计第 1 节增补 #7）；
// Memory 实现用于单测与本地无 Redis 环境（dev）。
package verifycode

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	TTL         = 5 * time.Minute
	MaxAttempts = 5
	CodeLen     = 6
)

var ErrTooManyAttempts = errors.New("verifycode: 尝试次数超限")

type Entry struct {
	Code     string
	ExpireAt time.Time
	Attempts int
}

// Store 验证码存取接口。phoneHash 为 HMAC 摘要（不落明文）。
type Store interface {
	// Put 写入验证码并重置尝试计数，TTL 5 分钟。
	Put(ctx context.Context, phoneHash, code string) error
	// Verify 校验验证码：正确则消费掉（一次性）；错误计入尝试，超上限返回 ErrTooManyAttempts。
	Verify(ctx context.Context, phoneHash, code string) (bool, error)
}

// MemoryStore 内存实现（单测 / 本地 dev；生产用 RedisStore）。
type MemoryStore struct {
	mu sync.Mutex
	m  map[string]Entry
}

func NewMemory() *MemoryStore { return &MemoryStore{m: map[string]Entry{}} }

func (s *MemoryStore) Put(ctx context.Context, phoneHash, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[phoneHash] = Entry{Code: code, ExpireAt: time.Now().Add(TTL)}
	return nil
}

func (s *MemoryStore) Verify(ctx context.Context, phoneHash, code string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[phoneHash]
	if !ok || time.Now().After(e.ExpireAt) {
		delete(s.m, phoneHash)
		return false, nil
	}
	if e.Attempts >= MaxAttempts {
		return false, ErrTooManyAttempts
	}
	if e.Code != code {
		e.Attempts++
		s.m[phoneHash] = e
		return false, nil
	}
	delete(s.m, phoneHash)
	return true, nil
}

// Generate 生成 6 位数字验证码。
func Generate() (string, error) {
	n, err := cryptoRandInt(1000000)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n), nil
}

func cryptoRandInt(max int64) (int64, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return 0, err
	}
	var v int64
	for _, c := range b {
		v = v<<8 | int64(c)
	}
	if v < 0 {
		v = -v
	}
	return v % max, nil
}
