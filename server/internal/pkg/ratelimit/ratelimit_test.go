package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryAllow_FirstWindow(t *testing.T) {
	l := NewMemory()
	ctx := context.Background()
	key := "sms:freq:h1"
	for i := int64(1); i <= 3; i++ {
		ok, err := l.Allow(ctx, key, 3, time.Minute)
		if err != nil || !ok {
			t.Fatalf("第 %d 次应允许: %v %v", i, ok, err)
		}
	}
	ok, _ := l.Allow(ctx, key, 3, time.Minute)
	if ok {
		t.Fatalf("第 4 次应拒绝")
	}
}

func TestMemoryAllow_WindowReset(t *testing.T) {
	l := NewMemory()
	ctx := context.Background()
	key := "sms:freq:h2"
	if _, err := l.Allow(ctx, key, 1, 10*time.Millisecond); err != nil {
		t.Fatalf("allow: %v", err)
	}
	if ok, _ := l.Allow(ctx, key, 1, 10*time.Millisecond); ok {
		t.Fatalf("窗口内第 2 次应拒绝")
	}
	time.Sleep(15 * time.Millisecond)
	if ok, _ := l.Allow(ctx, key, 1, 10*time.Millisecond); !ok {
		t.Fatalf("窗口过后应重新允许")
	}
}

func TestMemoryAllow_IndependentKeys(t *testing.T) {
	l := NewMemory()
	ctx := context.Background()
	if ok, _ := l.Allow(ctx, "a", 1, time.Minute); !ok {
		t.Fatalf("key a 第一次应允许")
	}
	if ok, _ := l.Allow(ctx, "b", 1, time.Minute); !ok {
		t.Fatalf("key b 不应受 key a 影响")
	}
}
