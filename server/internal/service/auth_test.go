package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/crypto"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/ratelimit"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/sms"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/verifycode"
)

// ---- 测试桩（纯内存，不依赖 DB/Redis）----

type fakeMembers struct {
	m      *model.Member
	s      *model.MemberSecure
	called int
}

func (f *fakeMembers) FindSecureByPhoneHash(hash string) (*model.Member, *model.MemberSecure, error) {
	f.called++
	if f.m == nil {
		return nil, nil, nil
	}
	return f.m, f.s, nil
}

type fakeByID struct{ m *model.Member }

func (f *fakeByID) FindByID(id int64) (*model.Member, error) {
	if f.m == nil || f.m.ID != id {
		return nil, errors.New("not found")
	}
	return f.m, nil
}

type fakeApps struct{ a *model.Application }

func (f *fakeApps) LatestByPhoneHash(hash string) (*model.Application, error) { return f.a, nil }

type fakeFails struct {
	m      map[string]int64
	window map[string]time.Time
}

func newFakeFails() *fakeFails {
	return &fakeFails{m: map[string]int64{}, window: map[string]time.Time{}}
}

func (f *fakeFails) Hit(ctx context.Context, key string, window time.Duration) (int64, error) {
	if exp, ok := f.window[key]; ok && time.Now().After(exp) {
		delete(f.m, key)
	}
	f.m[key]++
	if f.m[key] == 1 {
		f.window[key] = time.Now().Add(window)
	}
	return f.m[key], nil
}
func (f *fakeFails) Count(ctx context.Context, key string) (int64, error) {
	return f.m[key], nil
}
func (f *fakeFails) Reset(ctx context.Context, key string) error { delete(f.m, key); return nil }

type fakeRefresh struct{ m map[int64]string }

func newFakeRefresh() *fakeRefresh { return &fakeRefresh{m: map[int64]string{}} }

func (f *fakeRefresh) Get(ctx context.Context, id int64) (string, error) { return f.m[id], nil }
func (f *fakeRefresh) Set(ctx context.Context, id int64, jti string) error {
	f.m[id] = jti
	return nil
}
func (f *fakeRefresh) Delete(ctx context.Context, id int64) error { delete(f.m, id); return nil }

// ---- 构造 ----

const testSecret = "unit-secret"

func newAuth(t *testing.T, m *fakeMembers, byID MemberByID, apps *fakeApps) *AuthService {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	phone, err := crypto.NewPhoneCrypto(key)
	if err != nil {
		t.Fatalf("crypto: %v", err)
	}
	// 桩成员带真实密文（登录后 meVO 需解密出掩码号码）
	if m != nil && m.m != nil && m.s != nil {
		cipher, cerr := phone.Encrypt("13800001234")
		if cerr != nil {
			t.Fatalf("加密失败: %v", cerr)
		}
		m.s.PhoneCipher = cipher
	}
	return NewAuthService(
		m, byID, apps, verifycode.NewMemory(),
		sms.MockSender{}, ratelimit.NewMemory(), newFakeFails(), newFakeRefresh(),
		phone, true /*mock SMS*/, testSecret,
	)
}

func activeMember() *fakeMembers {
	return &fakeMembers{
		m: &model.Member{ID: 7, Nickname: "LU", Role: model.RoleMember, Status: model.MemberActive},
		s: &model.MemberSecure{MemberID: 7},
	}
}

// ---- 用例 ----

func TestSendCode_FreqLimitPer60s(t *testing.T) {
	s := newAuth(t, activeMember(), nil, &fakeApps{})
	ctx := context.Background()
	if err := s.SendCode(ctx, "13800001234", "1.2.3.4"); err != nil {
		t.Fatalf("第一次发送应成功: %v", err)
	}
	if err := s.SendCode(ctx, "13800001234", "1.2.3.4"); !errors.Is(err, code.ErrRateLimited) {
		t.Fatalf("60 秒内第二条应限流: %v", err)
	}
}

func TestSendCode_InvalidPhone(t *testing.T) {
	s := newAuth(t, activeMember(), nil, &fakeApps{})
	if err := s.SendCode(context.Background(), "123", ""); !errors.Is(err, code.ErrValidation) {
		t.Fatalf("非法号码应 1001: %v", err)
	}
}

func TestLogin_HappyPathMember(t *testing.T) {
	m := activeMember()
	s := newAuth(t, m, &fakeByID{m: m.m}, &fakeApps{})
	ctx := context.Background()
	_ = s.SendCode(ctx, "13800001234", "ip")
	res, err := s.Login(ctx, "13800001234", sms.MockVerifyCode, "ip")
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if res.Outcome != OutcomeMember || res.Tokens == nil || res.Member == nil {
		t.Fatalf("应返回 member + 令牌 + 资料: %+v", res)
	}
	if res.Member.MaskedPhone != "138****1234" {
		t.Fatalf("掩码号码不符: %s", res.Member.MaskedPhone)
	}
}

func TestLogin_WrongCodeIncrementsFails(t *testing.T) {
	m := activeMember()
	s := newAuth(t, m, &fakeByID{m: m.m}, &fakeApps{})
	ctx := context.Background()
	_ = s.SendCode(ctx, "13800001234", "ip")
	for i := 0; i < 4; i++ {
		if _, err := s.Login(ctx, "13800001234", "000000", "ip"); !errors.Is(err, code.ErrSmsCodeInvalid) {
			t.Fatalf("第 %d 次错码应 3001: %v", i+1, err)
		}
	}
	// 第 5 次失败后触发锁定
	if _, err := s.Login(ctx, "13800001234", "000000", "ip"); !errors.Is(err, code.ErrSmsCodeInvalid) {
		t.Fatalf("第 5 次仍应 3001: %v", err)
	}
	// 第 6 次：即使验证码正确也应被锁（4001），验证「锁定先于验证码校验」
	if _, err := s.Login(ctx, "13800001234", sms.MockVerifyCode, "ip"); !errors.Is(err, code.ErrRateLimited) {
		t.Fatalf("锁定后应 4001: %v", err)
	}
}

func TestLogin_PendingApplication(t *testing.T) {
	s := newAuth(t, &fakeMembers{}, nil, &fakeApps{a: &model.Application{
		ID: 3, Status: model.AppSubmitted,
	}})
	ctx := context.Background()
	_ = s.SendCode(ctx, "13800001234", "ip")
	res, err := s.Login(ctx, "13800001234", sms.MockVerifyCode, "ip")
	if err != nil {
		t.Fatalf("登录不应报错: %v", err)
	}
	if res.Outcome != OutcomeApplicationPending || res.Application == nil || res.Application.ID != 3 {
		t.Fatalf("应返回申请进度: %+v", res)
	}
	if res.Tokens != nil {
		t.Fatalf("非成员不得发令牌")
	}
}

func TestLogin_NoRecord(t *testing.T) {
	s := newAuth(t, &fakeMembers{}, nil, &fakeApps{})
	ctx := context.Background()
	_ = s.SendCode(ctx, "13800001234", "ip")
	res, err := s.Login(ctx, "13800001234", sms.MockVerifyCode, "ip")
	if err != nil || res.Outcome != OutcomeNoRecord {
		t.Fatalf("无记录应 no_record: %+v, %v", res, err)
	}
}

func TestLogin_DisabledMember(t *testing.T) {
	m := &fakeMembers{
		m: &model.Member{ID: 7, Status: model.MemberDisabled},
		s: &model.MemberSecure{},
	}
	s2 := newAuth(t, m, nil, &fakeApps{})
	ctx := context.Background()
	_ = s2.SendCode(ctx, "13800001234", "ip")
	if _, err := s2.Login(ctx, "13800001234", sms.MockVerifyCode, "ip"); !errors.Is(err, code.ErrNoPermission) {
		t.Fatalf("停用成员应 5002: %v", err)
	}
}

func TestRefresh_RotationAndReuseDetection(t *testing.T) {
	m := activeMember()
	s := newAuth(t, m, &fakeByID{m: m.m}, &fakeApps{})
	ctx := context.Background()
	_ = s.SendCode(ctx, "13800001234", "ip")
	res, _ := s.Login(ctx, "13800001234", sms.MockVerifyCode, "ip")

	// 第一次刷新成功（轮换）
	tk1, err := s.Refresh(ctx, res.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("刷新应成功: %v", err)
	}
	if tk1.AccessToken == "" || tk1.RefreshToken == res.Tokens.RefreshToken {
		t.Fatalf("刷新必须轮换出新的 refresh token")
	}
	// 旧 refresh 复用 → 2002 全会话吊销
	if _, err := s.Refresh(ctx, res.Tokens.RefreshToken); !errors.Is(err, code.ErrTokenReuse) {
		t.Fatalf("旧 refresh 复用应 2002: %v", err)
	}
	// 吊销后新 refresh 也失效 → 2001
	if _, err := s.Refresh(ctx, tk1.RefreshToken); !errors.Is(err, code.ErrUnauthorized) {
		t.Fatalf("吊销后新 refresh 应 2001: %v", err)
	}
}

func TestRefresh_AccessTokenRejected(t *testing.T) {
	m := activeMember()
	s := newAuth(t, m, &fakeByID{m: m.m}, &fakeApps{})
	ctx := context.Background()
	_ = s.SendCode(ctx, "13800001234", "ip")
	res, _ := s.Login(ctx, "13800001234", sms.MockVerifyCode, "ip")
	if _, err := s.Refresh(ctx, res.Tokens.AccessToken); !errors.Is(err, code.ErrUnauthorized) {
		t.Fatalf("access token 不得用于刷新: %v", err)
	}
}
