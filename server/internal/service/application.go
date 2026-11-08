package service

import (
	"context"
	"regexp"
	"time"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/crypto"
)

var inviteCodeRe = regexp.MustCompile(`^[A-Z0-9]{12}$`)

// ApplicationSubmitter 提交申请（事务在 repository 层）。
type ApplicationSubmitter interface {
	Submit(p SubmitParams) (int64, error)
}

// SubmitParams 与 repository 解耦的服务层参数。
type SubmitParams struct {
	InviteCode  string
	Nickname    string
	PhoneCipher string
	PhoneHash   string
}

// ApplicationLister 管理台申请列表。
type ApplicationLister interface {
	ListByStatus(status string, offset, limit int) ([]model.Application, int64, error)
}

// ApplicationReviewer 审核（通过=建成员事务；拒绝=回写原因）。
type ApplicationReviewer interface {
	FindByID(id int64) (*model.Application, error)
	Approve(in ApproveInput) error
	Reject(appID, adminID int64, reason, ip, ua string) error
}

// ApproveInput 与 repository 解耦的服务层入参。
type ApproveInput struct {
	Member   *model.Member
	Secure   *model.MemberSecure
	AdminID  int64
	Action   string
	TargetID int64
	IP       string
	UA       string
}

// ApplicationService 邀请与加入申请（细化设计 2.1 状态机）。
type ApplicationService struct {
	submit ApplicationSubmitter
	lister ApplicationLister
	review ApplicationReviewer
	phone  *crypto.PhoneCrypto
	Now    func() time.Time
}

func NewApplicationService(submit ApplicationSubmitter, lister ApplicationLister, review ApplicationReviewer, phone *crypto.PhoneCrypto) *ApplicationService {
	return &ApplicationService{submit: submit, lister: lister, review: review, phone: phone, Now: time.Now}
}

// Submit 提交申请：格式校验 → 归一与摘要 → 事务消费邀请码。
// 审核通过后以手机号验证码登录；结果通知（私信/推送）M5 接入。
func (s *ApplicationService) Submit(ctx context.Context, rawInvite, rawNickname, rawPhone string) (int64, string, error) {
	invite := normalizeInvite(rawInvite)
	if !inviteCodeRe.MatchString(invite) {
		return 0, "", code.ErrInviteInvalid
	}
	nickname, err := normalizeNickname(rawNickname)
	if err != nil {
		return 0, "", err
	}
	phone, err := crypto.Normalize(rawPhone)
	if err != nil {
		return 0, "", code.ErrValidation
	}
	cipher, err := s.phone.Encrypt(phone)
	if err != nil {
		return 0, "", err
	}
	hash := s.phone.Hash(phone)
	id, err := s.submit.Submit(SubmitParams{
		InviteCode: invite, Nickname: nickname, PhoneCipher: cipher, PhoneHash: hash,
	})
	return id, hash, err
}

// ListApplications 管理台列表（offset 分页）。
func (s *ApplicationService) ListApplications(status string, offset, limit int) ([]model.Application, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.lister.ListByStatus(status, offset, limit)
}

// GetApplication 管理台详情。
func (s *ApplicationService) GetApplication(id int64) (*model.Application, error) {
	return s.review.FindByID(id)
}

// Approve 通过：以申请行构造成员（角色 member，沿用称呼与手机号密文），事务生效。
func (s *ApplicationService) Approve(ctx context.Context, appID, adminID int64, ip, ua string) (int64, error) {
	app, err := s.review.FindByID(appID)
	if err != nil {
		return 0, err
	}
	if app.Status != model.AppSubmitted {
		return 0, code.ErrNotFound // 已结申请不可再审
	}
	m := &model.Member{
		Nickname:     app.Nickname,
		Role:         model.RoleMember,
		Status:       model.MemberActive,
		DefaultAnon:  true,
		DND:          true,
		NotifyNotice: true,
		NotifyHollow: true,
		JoinedAt:     s.Now(),
	}
	sec := &model.MemberSecure{PhoneCipher: app.PhoneCipher, PhoneHash: app.PhoneHash}
	in := ApproveInput{
		Member: m, Secure: sec, AdminID: adminID,
		Action: "admin.approve", TargetID: appID, IP: ip, UA: ua,
	}
	if err := s.review.Approve(in); err != nil {
		return 0, err
	}
	// TODO(M5): 审核结果私信主管理员 + 推送申请者（细化设计 7.1 通知矩阵）。
	return m.ID, nil
}

// RejectReasons 拒绝原因模板（管理员界面选取；产品口径待运营确认，细化设计 2.1）。
var RejectReasons = []string{
	"邀请码与申请信息不一致，请与管理员确认后重新申请",
	"申请信息不完整，请补充后重新提交",
	"暂不符合社群定位，感谢理解",
}

// Reject 拒绝（原因必须取自模板）。
func (s *ApplicationService) Reject(ctx context.Context, appID, adminID int64, reason, ip, ua string) error {
	if !validReason(reason) {
		return code.ErrValidation
	}
	return s.review.Reject(appID, adminID, reason, ip, ua)
}

func validReason(reason string) bool {
	for _, r := range RejectReasons {
		if r == reason {
			return true
		}
	}
	return false
}

// normalizeInvite 邀请码归一：去连字符与空格、转大写（展示格式 WMT-XXXX-XXXX）。
func normalizeInvite(code string) string {
	out := make([]byte, 0, len(code))
	for _, r := range code {
		switch r {
		case '-', ' ', '\t':
			continue
		}
		out = append(out, byte(upperByte(byte(r))))
	}
	return string(out)
}

func upperByte(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}

// normalizeNickname 称呼校验：1–8 字（产品规则），去首尾空格。
func normalizeNickname(n string) (string, error) {
	n = trimSpaces(n)
	r := []rune(n)
	if len(r) < 1 || len(r) > 8 {
		return "", code.ErrValidation
	}
	return n, nil
}
