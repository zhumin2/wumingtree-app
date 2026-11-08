package verifycode

import (
	"context"
	"errors"
	"testing"
)

func TestVerify_HappyPath(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if err := s.Put(ctx, "hash1", "123456"); err != nil {
		t.Fatalf("put: %v", err)
	}
	ok, err := s.Verify(ctx, "hash1", "123456")
	if err != nil || !ok {
		t.Fatalf("验证应通过: %v %v", ok, err)
	}
	// 一次性：消费后再验证失败
	ok, _ = s.Verify(ctx, "hash1", "123456")
	if ok {
		t.Fatalf("验证码应一次性消费")
	}
}

func TestVerify_AttemptLimit(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	_ = s.Put(ctx, "hash2", "123456")
	for i := 0; i < MaxAttempts; i++ {
		ok, err := s.Verify(ctx, "hash2", "000000")
		if ok || err != nil {
			t.Fatalf("第 %d 次错码不应通过: %v %v", i+1, ok, err)
		}
	}
	if _, err := s.Verify(ctx, "hash2", "123456"); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("超过 %d 次应返回 ErrTooManyAttempts, got %v", MaxAttempts, err)
	}
}

func TestGenerate_SixDigits(t *testing.T) {
	for i := 0; i < 100; i++ {
		c, err := Generate()
		if err != nil || len(c) != 6 {
			t.Fatalf("验证码应为 6 位: %s, %v", c, err)
		}
	}
}
