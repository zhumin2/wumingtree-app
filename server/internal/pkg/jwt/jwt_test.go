package jwt

import (
	"errors"
	"testing"
	"time"
)

const secret = "unit-test-secret"

func TestIssueParse_Roundtrip(t *testing.T) {
	tok, jti, err := Issue(secret, 42, "member", TypAccess, time.Hour)
	if err != nil || jti == "" {
		t.Fatalf("签发失败: %v", err)
	}
	claims, err := Parse(secret, tok)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if claims.Subject != "42" || claims.Role != "member" || claims.Typ != TypAccess || claims.JTI != jti {
		t.Fatalf("载荷不符: %+v", claims)
	}
}

func TestParse_WrongSecret(t *testing.T) {
	tok, _, _ := Issue(secret, 1, "member", TypAccess, time.Hour)
	if _, err := Parse("other-secret", tok); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("want ErrTokenInvalid, got %v", err)
	}
}

func TestParse_Expired(t *testing.T) {
	tok, _, _ := Issue(secret, 1, "member", TypAccess, -time.Minute)
	if _, err := Parse(secret, tok); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("want ErrTokenExpired, got %v", err)
	}
}

func TestParse_RefreshTypAccepted_AccessTypRequiredByMiddleware(t *testing.T) {
	tok, _, _ := Issue(secret, 1, "member", TypRefresh, time.Hour)
	claims, err := Parse(secret, tok)
	if err != nil || claims.Typ != TypRefresh {
		t.Fatalf("refresh 应可解析（拒绝逻辑在 middleware 层）: %v", err)
	}
}

func TestTTLMatchDesign(t *testing.T) {
	// 架构设计 3.3：access 7 天 + refresh 30 天。
	if AccessTTL != 7*24*time.Hour || RefreshTTL != 30*24*time.Hour {
		t.Fatalf("令牌时效与设计不符")
	}
}
