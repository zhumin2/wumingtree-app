package code

import (
	"errors"
	"fmt"
	"testing"
)

func TestOf_SentinelDirect(t *testing.T) {
	got, msg := Of(ErrQuotaFull)
	if got != CodeQuotaFull || msg != ErrQuotaFull.Message {
		t.Fatalf("want (%d,%s), got (%d,%s)", CodeQuotaFull, ErrQuotaFull.Message, got, msg)
	}
}

func TestOf_WrappedSentinel(t *testing.T) {
	wrapped := fmt.Errorf("报名占位失败: %w", ErrQuotaFull)
	got, _ := Of(wrapped)
	if got != CodeQuotaFull {
		t.Fatalf("want %d, got %d", CodeQuotaFull, got)
	}
}

func TestOf_UnknownErrorMapsToInternal(t *testing.T) {
	got, msg := Of(errors.New("敏感内部堆栈信息"))
	if got != CodeInternal {
		t.Fatalf("want %d, got %d", CodeInternal, got)
	}
	if msg != ErrInternal.Message {
		t.Fatalf("未知错误必须返回通用文案，不得泄露内部信息: %s", msg)
	}
}

func TestErrorCodesMatchDesignDoc(t *testing.T) {
	// 与《技术架构设计》3.2 / 《细化设计》第 3 节的数字对齐；改这里必须先改文档。
	cases := map[*BizError]int64{
		ErrValidation:      1001,
		ErrUnauthorized:    2001,
		ErrTokenReuse:      2002,
		ErrSmsCodeInvalid:  3001,
		ErrInviteInvalid:   3002,
		ErrRateLimited:     4001,
		ErrAuditRejected:   4101,
		ErrQuotaFull:       4201,
		ErrDuplicateEnroll: 4202,
		ErrPhoneDuplicate:  4203,
		ErrConsentRequired: 4204,
		ErrNotFound:        5001,
		ErrNoPermission:    5002,
		ErrInternal:        9000,
		ErrDependency:      9001,
	}
	for e, want := range cases {
		if e.Code != want {
			t.Fatalf("%s: want %d, got %d", e.Message, want, e.Code)
		}
	}
}
