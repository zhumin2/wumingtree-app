package idempot

import (
	"context"
	"testing"
	"time"
)

func TestAcquire_FirstWins(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	first, err := s.Acquire(ctx, "idem:t1", time.Hour)
	if err != nil || !first {
		t.Fatalf("首次应抢占成功: %v %v", first, err)
	}
	second, _ := s.Acquire(ctx, "idem:t1", time.Hour)
	if second {
		t.Fatalf("重复抢占应返回 false")
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if _, err := s.Acquire(ctx, "idem:t2", time.Hour); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := s.Save(ctx, "idem:t2", []byte(`{"id":7}`), time.Hour); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, ok, err := s.Load(ctx, "idem:t2")
	if err != nil || !ok || string(got) != `{"id":7}` {
		t.Fatalf("load: %s %v %v", got, ok, err)
	}
}

func TestLoad_BeforeSave_ReturnsNotReady(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	_, _ = s.Acquire(ctx, "idem:t3", time.Hour)
	_, ok, err := s.Load(ctx, "idem:t3")
	if err != nil || ok {
		t.Fatalf("Save 前应返回 (nil,false,nil): %v %v", ok, err)
	}
}

func TestAcquire_KeyExpiry(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	_, _ = s.Acquire(ctx, "idem:t4", 10*time.Millisecond)
	time.Sleep(15 * time.Millisecond)
	first, _ := s.Acquire(ctx, "idem:t4", time.Hour)
	if !first {
		t.Fatalf("过期后应可重新抢占")
	}
}
