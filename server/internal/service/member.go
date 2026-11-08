package service

import (
	"context"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/crypto"
)

// MeReader 按 ID 取成员。
type MeReader interface {
	FindByID(id int64) (*model.Member, error)
}

// ProfileUpdater 资料更新。
type ProfileUpdater interface {
	UpdateProfile(id int64, p MemberPatch) error
}

// Counters 我的页三项统计（报名 / 树洞 / 作品）。
type Counters interface {
	CountEnrollments(memberID int64) (int64, error)
	CountPosts(memberID int64) (int64, error)
	CountWorks(memberID int64) (int64, error)
}

// SecureCipherReader 本人手机号密文读取（掩码展示用）。
type SecureCipherReader interface {
	SecureCipherByMemberID(memberID int64) (string, error)
}

// MemberService 我的模块。
type MemberService struct {
	byID   MeReader
	cipher SecureCipherReader
	upd    ProfileUpdater
	counts Counters
	badges BadgeStore
	phone  *crypto.PhoneCrypto
}

func NewMemberService(byID MeReader, cipher SecureCipherReader, upd ProfileUpdater, counts Counters, badges BadgeStore, phone *crypto.PhoneCrypto) *MemberService {
	return &MemberService{byID: byID, cipher: cipher, upd: upd, counts: counts, badges: badges, phone: phone}
}

// Me 本人资料（含掩码手机号；明文任何接口不出现——红线）。
func (s *MemberService) Me(ctx context.Context, memberID int64) (*MeVO, error) {
	m, err := s.byID.FindByID(memberID)
	if err != nil {
		return nil, err
	}
	vo := &MeVO{
		ID: m.ID, Nickname: m.Nickname, Role: m.Role,
		DefaultAnon: m.DefaultAnon, DND: m.DND,
		NotifyNotice: m.NotifyNotice, NotifyHollow: m.NotifyHollow,
		JoinedAt: m.JoinedAt,
	}
	if s.phone != nil && s.cipher != nil {
		if cipher, err := s.cipher.SecureCipherByMemberID(memberID); err == nil && cipher != "" {
			if plain, err := s.phone.Decrypt(cipher); err == nil {
				vo.MaskedPhone, _ = crypto.Mask(plain)
			}
		}
	}
	return vo, nil
}

// UpdateMe 更新昵称 / 默认匿名 / 免打扰 / 提醒开关。
func (s *MemberService) UpdateMe(ctx context.Context, memberID int64, p MemberPatch) error {
	if p.Nickname != nil {
		n, err := normalizeNickname(*p.Nickname)
		if err != nil {
			return err
		}
		p.Nickname = &n
	}
	return s.upd.UpdateProfile(memberID, p)
}

// Badges 红点（架构设计 3.2 红线：仅布尔，无聚合计数）。
// 存储为位标记（Redis/Memory），有人回应帖子等事件置位（M2 hollow 接线）。
func (s *MemberService) Badges(ctx context.Context, memberID int64) map[string]bool {
	out := map[string]bool{"notice": false, "hollow": false, "dm": false}
	if s.badges == nil {
		return out
	}
	for _, k := range []string{"notice", "hollow", "dm"} {
		v, err := s.badges.Read(ctx, memberID, k)
		if err != nil || v == false {
			out[k] = false
			continue
		}
		out[k] = v
	}
	return out
}

// SummaryVO 我的页统计。
type SummaryVO struct {
	Enrollments int64 `json:"enrollments"`
	Posts       int64 `json:"posts"`
	Works       int64 `json:"works"`
}

// Summary 我的页三项统计。
func (s *MemberService) Summary(ctx context.Context, memberID int64) (*SummaryVO, error) {
	if s.counts == nil {
		return &SummaryVO{}, nil
	}
	var v SummaryVO
	var err error
	if v.Enrollments, err = s.counts.CountEnrollments(memberID); err != nil {
		return nil, err
	}
	if v.Posts, err = s.counts.CountPosts(memberID); err != nil {
		return nil, err
	}
	if v.Works, err = s.counts.CountWorks(memberID); err != nil {
		return nil, err
	}
	return &v, nil
}

// 防 code/model 误删引用（编译期哨兵）。
var (
	_ = code.CodeOK
	_ = model.MemberActive
)
