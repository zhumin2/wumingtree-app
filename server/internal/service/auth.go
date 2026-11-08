// Package service 全部业务规则（架构设计 3.1）：校验、频控、事务编排、权限判断。
// 依赖均为接口，纯逻辑可脱离 DB 单测； repository / redis 实现在 repository 层。
package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/crypto"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/jwt"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/ratelimit"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/sms"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/verifycode"
)

// ---- 服务依赖接口（消费方定义；GORM/Redis 实现在 repository 层）----

// MemberFinder 按摘要取成员与隔离记录。
type MemberFinder interface {
	FindSecureByPhoneHash(hash string) (*model.Member, *model.MemberSecure, error)
}

// MemberPatch 资料更新项（nil = 不更新）。
type MemberPatch struct {
	Nickname     *string
	DefaultAnon  *bool
	DND          *bool
	NotifyNotice *bool
	NotifyHollow *bool
}

// ApplicationLatestFinder 取同摘要最近申请。
type ApplicationLatestFinder interface {
	LatestByPhoneHash(hash string) (*model.Application, error)
}

// MemberByID 按 ID 取成员（刷新令牌时校验状态）。
type MemberByID interface {
	FindByID(id int64) (*model.Member, error)
}

// RefreshStore 刷新令牌轮换状态（当前会话模型：每成员一份 jti，架构设计 3.3）。
// 复用检测：传入 jti 与存储不符 → 吊销（Delete）并拒绝（2002）。
type RefreshStore interface {
	Get(ctx context.Context, memberID int64) (string, error)
	Set(ctx context.Context, memberID int64, jti string) error
	Delete(ctx context.Context, memberID int64) error
}

// FailCounter 登录失败计数（同号连续失败 5 次锁 15 分钟，细化设计 6）。
type FailCounter interface {
	Hit(ctx context.Context, key string, window time.Duration) (int64, error)
	Count(ctx context.Context, key string) (int64, error)
	Reset(ctx context.Context, key string) error
}

// ---- VO ----

// Tokens 令牌对。
type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// MeVO 我的资料。红线：任何接口不出现明文手机号；maskedPhone 仅为展示串。
type MeVO struct {
	ID           int64     `json:"id"`
	Nickname     string    `json:"nickname"`
	Role         string    `json:"role"`
	DefaultAnon  bool      `json:"defaultAnon"`
	DND          bool      `json:"dnd"`
	NotifyNotice bool      `json:"notifyNotice"`
	NotifyHollow bool      `json:"notifyHollow"`
	JoinedAt     time.Time `json:"joinedAt"`
	MaskedPhone  string    `json:"maskedPhone"`
}

// ApplicationVO 申请进度视图（登录 outcome 附带）。
type ApplicationVO struct {
	ID           int64      `json:"id"`
	Status       string     `json:"status"`
	RejectReason string     `json:"rejectReason,omitempty"`
	SubmittedAt  time.Time  `json:"submittedAt"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
}

// LoginOutcome 登录结果类型（openapi LoginResult.outcome；no_record 引导去 gate）。
const (
	OutcomeMember             = "member"
	OutcomeApplicationPending = "application_pending"
	OutcomeApplicationReject  = "application_rejected"
	OutcomeNoRecord           = "no_record"
)

// LoginResult 登录结果。
type LoginResult struct {
	Outcome     string         `json:"outcome"`
	Tokens      *Tokens        `json:"tokens,omitempty"`
	Member      *MeVO          `json:"member,omitempty"`
	Application *ApplicationVO `json:"application,omitempty"`
}

// ---- AuthService ----

// AuthService 登录与会话。
type AuthService struct {
	members MemberFinder
	byID    MemberByID
	apps    ApplicationLatestFinder
	codes   verifycode.Store
	sender  sms.Sender
	limiter ratelimit.Limiter
	fails   FailCounter
	refresh RefreshStore
	phone   *crypto.PhoneCrypto
	mockSMS bool
	secret  string
	Now     func() time.Time // 可注入时钟（测试禁用 now-5min 构造，参考架构测试坑）
}

func NewAuthService(
	members MemberFinder, byID MemberByID, apps ApplicationLatestFinder, codes verifycode.Store,
	sender sms.Sender, limiter ratelimit.Limiter, fails FailCounter, refresh RefreshStore,
	phone *crypto.PhoneCrypto, mockSMS bool, secret string,
) *AuthService {
	return &AuthService{
		members: members, byID: byID, apps: apps, codes: codes, sender: sender,
		limiter: limiter, fails: fails, refresh: refresh, phone: phone,
		mockSMS: mockSMS, secret: secret, Now: time.Now,
	}
}

// SendCode 发送验证码：60s/条、5 条/天/号、20 条/天/IP（细化设计 6）。
func (s *AuthService) SendCode(ctx context.Context, rawPhone, ip string) error {
	phone, err := crypto.Normalize(rawPhone)
	if err != nil {
		return code.ErrValidation
	}
	hash := s.phone.Hash(phone)
	// 频控：60s 1 条
	if ok, _ := s.limiter.Allow(ctx, "sms:freq:"+hash, 1, time.Minute); !ok {
		return code.ErrRateLimited
	}
	// 频控：24h 5 条
	if ok, _ := s.limiter.Allow(ctx, "sms:day:"+hash, 5, 24*time.Hour); !ok {
		return code.ErrRateLimited
	}
	// 频控：同 IP 24h 20 条（防批量）
	if ip != "" {
		if ok, _ := s.limiter.Allow(ctx, "sms:ip:"+ip, 20, 24*time.Hour); !ok {
			return code.ErrRateLimited
		}
	}
	vc := sms.MockVerifyCode
	if !s.mockSMS {
		if vc, err = verifycode.Generate(); err != nil {
			return err
		}
	}
	if err := s.codes.Put(ctx, hash, vc); err != nil {
		return err
	}
	return s.sender.SendVerifyCode(ctx, phone, vc)
}

// Login 手机号 + 验证码登录：成员 / 申请状态 / 无记录三路返回（细化设计 2.1）。
func (s *AuthService) Login(ctx context.Context, rawPhone, inputCode, ip string) (*LoginResult, error) {
	phone, err := crypto.Normalize(rawPhone)
	if err != nil {
		return nil, code.ErrValidation
	}
	hash := s.phone.Hash(phone)
	failKey := "login:fail:" + hash

	// 锁定检查（连续失败 5 次锁 15 分钟）
	if n, _ := s.fails.Count(ctx, failKey); n >= 5 {
		return nil, code.ErrRateLimited
	}

	ok, verr := s.codes.Verify(ctx, hash, inputCode)
	if verr != nil && !errors.Is(verr, verifycode.ErrTooManyAttempts) {
		return nil, verr
	}
	if !ok {
		s.fails.Hit(ctx, failKey, 15*time.Minute)
		return nil, code.ErrSmsCodeInvalid
	}
	s.fails.Reset(ctx, failKey)

	m, sec, err := s.members.FindSecureByPhoneHash(hash)
	if err != nil {
		return nil, err
	}
	if m == nil {
		// 非成员：申请状态或引导去 gate
		app, err := s.apps.LatestByPhoneHash(hash)
		if err != nil {
			return nil, err
		}
		res := &LoginResult{Outcome: OutcomeNoRecord}
		if app != nil {
			res.Outcome = OutcomeApplicationPending
			if app.Status == model.AppRejected {
				res.Outcome = OutcomeApplicationReject
			}
			res.Application = applicationVO(app)
		}
		return res, nil
	}
	if m.Status != model.MemberActive {
		return nil, code.ErrNoPermission
	}
	tokens, err := s.issueTokens(ctx, m)
	if err != nil {
		return nil, err
	}
	mv, err := s.meVO(m, sec)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Outcome: OutcomeMember, Tokens: tokens, Member: mv}, nil
}

// Refresh 刷新令牌（一次性轮换 + 复用检测，架构设计 3.3）。
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*Tokens, error) {
	claims, err := jwt.Parse(s.secret, refreshToken)
	if err != nil {
		return nil, code.ErrUnauthorized
	}
	if claims.Typ != jwt.TypRefresh {
		return nil, code.ErrUnauthorized
	}
	mid, perr := strconv.ParseInt(claims.Subject, 10, 64)
	if perr != nil {
		return nil, code.ErrUnauthorized
	}
	stored, err := s.refresh.Get(ctx, mid)
	if err != nil {
		return nil, err
	}
	if stored == "" {
		return nil, code.ErrUnauthorized // 会话不存在（已过期或已登出）
	}
	if stored != claims.JTI {
		// 复用 = 可能泄露：吊销该成员全部会话（2002）
		_ = s.refresh.Delete(ctx, mid)
		return nil, code.ErrTokenReuse
	}
	m, err := s.byID.FindByID(mid)
	if err != nil {
		return nil, code.ErrUnauthorized
	}
	if m.Status != model.MemberActive {
		return nil, code.ErrNoPermission
	}
	// 轮换：新令牌对 + 覆盖 jti（旧 refresh 自此失效）
	return s.issueTokens(ctx, m)
}

// issueTokens 签发并登记 refresh jti。
func (s *AuthService) issueTokens(ctx context.Context, m *model.Member) (*Tokens, error) {
	access, _, err := jwt.Issue(s.secret, m.ID, m.Role, jwt.TypAccess, jwt.AccessTTL)
	if err != nil {
		return nil, err
	}
	refresh, jti, err := jwt.Issue(s.secret, m.ID, m.Role, jwt.TypRefresh, jwt.RefreshTTL)
	if err != nil {
		return nil, err
	}
	if err := s.refresh.Set(ctx, m.ID, jti); err != nil {
		return nil, err
	}
	return &Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *AuthService) meVO(m *model.Member, sec *model.MemberSecure) (*MeVO, error) {
	masked := ""
	if sec != nil {
		plain, err := s.phone.Decrypt(sec.PhoneCipher)
		if err != nil {
			return nil, err
		}
		masked, err = crypto.Mask(plain)
		if err != nil {
			return nil, err
		}
	}
	return &MeVO{
		ID: m.ID, Nickname: m.Nickname, Role: m.Role,
		DefaultAnon: m.DefaultAnon, DND: m.DND,
		NotifyNotice: m.NotifyNotice, NotifyHollow: m.NotifyHollow,
		JoinedAt: m.JoinedAt, MaskedPhone: masked,
	}, nil
}

func applicationVO(a *model.Application) *ApplicationVO {
	v := &ApplicationVO{
		ID: a.ID, Status: a.Status, SubmittedAt: a.CreatedAt, ReviewedAt: a.ReviewedAt,
	}
	if a.RejectReason != nil {
		v.RejectReason = *a.RejectReason
	}
	return v
}
