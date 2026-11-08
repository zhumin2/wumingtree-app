package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/audit"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/idempot"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/ratelimit"
)

// ---- 测试桩（纯内存）----

type fakePostStore struct {
	seq     int64
	posts   map[int64]*PostRow
	members MemberByID
}

func newFakePosts(members MemberByID) *fakePostStore {
	return &fakePostStore{posts: map[int64]*PostRow{}, members: members}
}

func (f *fakePostStore) Create(p *model.Post) error {
	f.seq++
	p.ID = f.seq
	row := &PostRow{Post: *p}
	if f.members != nil {
		if m, err := f.members.FindByID(p.AuthorID); err == nil && m != nil {
			row.AuthorNickname = m.Nickname
		}
	}
	f.posts[p.ID] = row
	return nil
}

func (f *fakePostStore) FindByID(id int64) (*PostRow, error) {
	if r, ok := f.posts[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, nil
}

func (f *fakePostStore) List(mood string, page Page) ([]PostRow, string, error) {
	var out []PostRow
	for _, r := range f.posts {
		if r.Post.Status != model.PostPublished {
			continue
		}
		if mood != "" && r.Post.Mood != mood {
			continue
		}
		out = append(out, *r)
	}
	return out, "", nil
}

func (f *fakePostStore) ListByAuthor(authorID int64, page Page) ([]PostRow, string, error) {
	var out []PostRow
	for _, r := range f.posts {
		if r.Post.AuthorID == authorID && r.Post.Status == model.PostPublished {
			out = append(out, *r)
		}
	}
	return out, "", nil
}

func (f *fakePostStore) UpdateStatus(id int64, status, removedReason string) error {
	if r, ok := f.posts[id]; ok {
		r.Post.Status = status
		r.Post.RemovedReason = &removedReason
	}
	return nil
}

type fakeFloorStore struct {
	seq    int64
	floors map[int64]*model.Floor
}

func newFakeFloors() *fakeFloorStore { return &fakeFloorStore{floors: map[int64]*model.Floor{}} }

func (f *fakeFloorStore) Create(fl *model.Floor) error {
	f.seq++
	fl.ID = f.seq
	f.floors[fl.ID] = fl
	return nil
}

func (f *fakeFloorStore) FindByID(id int64) (*model.Floor, error) {
	if fl, ok := f.floors[id]; ok {
		cp := *fl
		return &cp, nil
	}
	return nil, nil
}

func (f *fakeFloorStore) CountPublished(postID int64) (int64, error) {
	var n int64
	for _, fl := range f.floors {
		if fl.PostID == postID && fl.Status == model.FloorPublished {
			n++
		}
	}
	return n, nil
}

func (f *fakeFloorStore) CountPublishedByPosts(postIDs []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	for _, id := range postIDs {
		out[id], _ = f.CountPublished(id)
	}
	return out, nil
}

func (f *fakeFloorStore) ListByPost(postID int64, page Page) ([]FloorRow, string, error) {
	return nil, "", nil
}

type fakeReactionStore struct {
	// key: targetType|targetID|memberID|type
	active map[string]bool
}

func newFakeReactions() *fakeReactionStore { return &fakeReactionStore{active: map[string]bool{}} }

func key(memberID int64, targetType string, targetID int64, rtype string) string {
	return targetType + "|" + int64Key(targetID) + "|" + int64Key(memberID) + "|" + rtype
}

func (f *fakeReactionStore) Toggle(memberID int64, targetType string, targetID int64, rtype string) (bool, error) {
	k := key(memberID, targetType, targetID, rtype)
	if f.active[k] {
		f.active[k] = false
		return false, nil
	}
	f.active[k] = true
	return true, nil
}

func (f *fakeReactionStore) Count(targetType string, targetID int64, rtype string) (int64, error) {
	var n int64
	for k, v := range f.active {
		if v && strings.HasPrefix(k, targetType+"|"+int64Key(targetID)+"|") && strings.HasSuffix(k, "|"+rtype) {
			n++
		}
	}
	return n, nil
}

func (f *fakeReactionStore) Mine(memberID int64, targetType string, targetID int64) ([]string, error) {
	var out []string
	for k, v := range f.active {
		if v && strings.HasPrefix(k, targetType+"|"+int64Key(targetID)+"|") && strings.HasSuffix(k, "|"+int64Key(memberID)) {
			parts := strings.Split(k, "|")
			out = append(out, parts[len(parts)-1])
		}
	}
	return out, nil
}

func (f *fakeReactionStore) MineByTargets(memberID int64, targetType string, ids []int64) (map[int64][]string, error) {
	out := map[int64][]string{}
	for _, id := range ids {
		l, _ := f.Mine(memberID, targetType, id)
		if len(l) > 0 {
			out[id] = l
		}
	}
	return out, nil
}

func (f *fakeReactionStore) CountByTargets(targetType string, ids []int64) (map[int64]map[string]int64, error) {
	out := map[int64]map[string]int64{}
	for _, id := range ids {
		for _, rt := range []string{"hug", "me_too", "helpful"} {
			n, _ := f.Count(targetType, id, rt)
			if n > 0 {
				if out[id] == nil {
					out[id] = map[string]int64{}
				}
				out[id][rt] = n
			}
		}
	}
	return out, nil
}

type fakeReportStore struct{ seq int64 }

func (f *fakeReportStore) Create(rep *model.Report) error { return nil }
func (f *fakeReportStore) ListByStatus(status string, offset, limit int) ([]model.Report, int64, error) {
	return nil, 0, nil
}
func (f *fakeReportStore) Handle(reportID, adminID int64, action, note string) error { return nil }

type fakeBadges struct{ m map[int64]map[string]bool }

func newFakeBadges() *fakeBadges { return &fakeBadges{m: map[int64]map[string]bool{}} }

func (f *fakeBadges) Mark(ctx context.Context, memberID int64, kind string) error {
	if f.m[memberID] == nil {
		f.m[memberID] = map[string]bool{}
	}
	f.m[memberID][kind] = true
	return nil
}
func (f *fakeBadges) Clear(ctx context.Context, memberID int64, kind string) error { return nil }
func (f *fakeBadges) Read(ctx context.Context, memberID int64, kind string) (bool, error) {
	return f.m[memberID][kind], nil
}

type fakeAudits struct{ actions []string }

func (f *fakeAudits) Append(actorID int64, action, targetType string, targetID int64, ip, ua string) error {
	f.actions = append(f.actions, action)
	return nil
}

// ---- 构造 ----

func newHollow(t *testing.T, members MemberByID, audits *fakeAudits) *HollowService {
	t.Helper()
	return NewHollowService(
		newFakePosts(members), newFakeFloors(), newFakeReactions(),
		members, audits, audit.MockAuditor{},
		ratelimit.NewMemory(), idempot.NewMemory(), newFakeBadges(),
	)
}

var ctxBg = context.Background()

func anonAuthor(t *testing.T) *fakeByID {
	return &fakeByID{m: &model.Member{ID: 5, Nickname: "阿柚", Role: model.RoleMember, Status: model.MemberActive}}
}

// ---- 用例 ----

func TestCreatePost_IdempotencyBeforeRateLimit(t *testing.T) {
	// 铁律验证：同 token 重试在频控耗尽后仍返回首次结果。
	s := newHollow(t, anonAuthor(t), &fakeAudits{})
	// 首次携 token 发帖成功
	first, err := s.CreatePost(ctxBg, 5, "share", "第一句", nil, true, "tok-retry-1")
	if err != nil {
		t.Fatalf("首次发帖不应失败: %v", err)
	}
	// 打满当日频控（共 10 条）
	for i := 0; i < 9; i++ {
		_, err = s.CreatePost(ctxBg, 5, "share", "凑数", nil, true, "")
		if err != nil {
			t.Fatalf("第 %d 次发帖不应失败: %v", i+2, err)
		}
	}
	// 超限后普通发帖 4001
	if _, err = s.CreatePost(ctxBg, 5, "share", "超限了", nil, true, ""); !strings.Contains(err.Error(), code.ErrRateLimited.Message) {
		t.Fatalf("第 11 次应 4001: %v", err)
	}
	// 携带 token 的重试：幂等先于频控，仍返回首次结果
	retry, err := s.CreatePost(ctxBg, 5, "share", "第一句", nil, true, "tok-retry-1")
	if err != nil {
		t.Fatalf("同 token 重试不应被频控拦截: %v", err)
	}
	if retry.ID != first.ID {
		t.Fatalf("重试应返回首次结果: first=%d retry=%d", first.ID, retry.ID)
	}
	retry2, _ := s.CreatePost(ctxBg, 5, "share", "第一句", nil, true, "tok-retry-1")
	if retry2.ID != first.ID {
		t.Fatalf("token 复用应再次返回首次结果")
	}
}

func TestCreatePost_CrisisNotRejected(t *testing.T) {
	audits := &fakeAudits{}
	s := newHollow(t, anonAuthor(t), audits)
	vo, err := s.CreatePost(ctxBg, 5, "tired", "最近很累", strPtr("有时候真的不想活了"), true, "")
	if err != nil {
		t.Fatalf("危机词必须不拦截（红线）: %v", err)
	}
	if vo.ID == 0 {
		t.Fatalf("危机内容应正常发布")
	}
	found := false
	for _, a := range audits.actions {
		if a == "crisis.alert" {
			found = true
		}
	}
	if !found {
		t.Fatalf("危机内容应有高优告警记录")
	}
}

func TestCreatePost_RejectWord(t *testing.T) {
	s := newHollow(t, anonAuthor(t), &fakeAudits{})
	_, err := s.CreatePost(ctxBg, 5, "share", "低价代购加微信", nil, true, "")
	if err == nil || err.Error() != code.ErrAuditRejected.Message {
		t.Fatalf("违规词应 4101: %v", err)
	}
}

func TestGetPost_AnonymousVOHasNoAuthorLeak(t *testing.T) {
	s := newHollow(t, anonAuthor(t), &fakeAudits{})
	vo, err := s.CreatePost(ctxBg, 5, "tired", "还是没敢跟老师说", nil, true, "")
	if err != nil {
		t.Fatalf("发帖: %v", err)
	}
	if vo.DisplayName != "一位家长" {
		t.Fatalf("匿名显示名不符: %s", vo.DisplayName)
	}
	detail, err := s.GetPost(ctxBg, 5, vo.ID)
	if err != nil {
		t.Fatalf("详情: %v", err)
	}
	payload, _ := json.Marshal(detail)
	if strings.Contains(string(payload), "author") || strings.Contains(string(payload), "阿柚") {
		t.Fatalf("匿名详情响应泄露作者信息: %s", payload)
	}
}

func TestGetPost_RealNameShowsNickname(t *testing.T) {
	s := newHollow(t, anonAuthor(t), &fakeAudits{})
	vo, _ := s.CreatePost(ctxBg, 5, "ask", "有人居家过整学期吗", nil, false, "")
	detail, _ := s.GetPost(ctxBg, 5, vo.ID)
	if detail.Post.DisplayName != "阿柚" {
		t.Fatalf("实名帖应显示昵称: %s", detail.Post.DisplayName)
	}
}

func TestCreateFloor_DepthAndValidation(t *testing.T) {
	s := newHollow(t, anonAuthor(t), &fakeAudits{})
	post, _ := s.CreatePost(ctxBg, 5, "share", "第一句", nil, true, "")
	f1, err := s.CreateFloor(ctxBg, 5, post.ID, "我先接住你", nil, true, "")
	if err != nil || f1.Depth != 1 {
		t.Fatalf("一层楼: %v %d", err, f1.Depth)
	}
	f2, err := s.CreateFloor(ctxBg, 5, post.ID, "我也是", &f1.ID, true, "")
	if err != nil || f2.Depth != 2 {
		t.Fatalf("楼中楼: %v %d", err, f2.Depth)
	}
	// 三层不允许：对 2 层回应 → 1001
	if _, err := s.CreateFloor(ctxBg, 5, post.ID, "三层", &f2.ID, true, ""); err == nil {
		t.Fatalf("楼中楼仅一层，三层应拒绝")
	}
}

func TestTogglePostReaction(t *testing.T) {
	s := newHollow(t, anonAuthor(t), &fakeAudits{})
	post, _ := s.CreatePost(ctxBg, 5, "share", "第一句", nil, true, "")
	st, err := s.TogglePostReaction(ctxBg, 5, post.ID, "hug")
	if err != nil || !st.Active || st.Count != 1 {
		t.Fatalf("首次 hug 应 active+1: %+v %v", st, err)
	}
	st, _ = s.TogglePostReaction(ctxBg, 5, post.ID, "hug")
	if st.Active || st.Count != 0 {
		t.Fatalf("再次 hug 应取消: %+v", st)
	}
	// 负向类型不存在（红线）
	if _, err := s.TogglePostReaction(ctxBg, 5, post.ID, "dislike"); err == nil {
		t.Fatalf("负向类型必须拒绝")
	}
}

func TestDeletePost_Permissions(t *testing.T) {
	posts := newFakePosts(anonAuthor(t))
	s := newHollow(t, anonAuthor(t), &fakeAudits{})
	post, _ := s.CreatePost(ctxBg, 5, "share", "我的帖子", nil, true, "")

	// 非作者非管理员 → 5002
	other := &fakeByID{m: &model.Member{ID: 9, Nickname: "别人", Role: model.RoleMember, Status: model.MemberActive}}
	s2 := NewHollowService(s.posts, s.floors, s.react, other, &fakeAudits{}, audit.MockAuditor{}, ratelimit.NewMemory(), idempot.NewMemory(), newFakeBadges())
	if err := s2.DeletePost(ctxBg, 9, post.ID); err == nil || err.Error() != code.ErrNoPermission.Message {
		t.Fatalf("非作者应 5002: %v", err)
	}

	// 管理员可删
	s3 := NewHollowService(s.posts, s.floors, s.react, &fakeByID{m: &model.Member{ID: 1, Nickname: "管理员", Role: model.RoleAdmin, Status: model.MemberActive}}, &fakeAudits{}, audit.MockAuditor{}, ratelimit.NewMemory(), idempot.NewMemory(), newFakeBadges())
	if err := s3.DeletePost(ctxBg, 1, post.ID); err != nil {
		t.Fatalf("管理员删帖: %v", err)
	}
	if _, err := s.GetPost(ctxBg, 5, post.ID); err == nil {
		t.Fatalf("删帖后详情应 5001")
	}
	_ = posts
}

func TestMyPosts_ShowsOwnNicknameForAnonymous(t *testing.T) {
	s := newHollow(t, anonAuthor(t), &fakeAudits{})
	if _, err := s.CreatePost(ctxBg, 5, "tired", "匿名帖", nil, true, ""); err != nil {
		t.Fatalf("发帖: %v", err)
	}
	page, err := s.MyPosts(ctxBg, 5, Page{})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("我的树洞应含匿名帖: %v %v", page, err)
	}
	if page.Items[0].DisplayName != "阿柚" {
		t.Fatalf("我的树洞中匿名帖应显示为自己的记录: %s", page.Items[0].DisplayName)
	}
}

func TestHollowBadgeOnReply(t *testing.T) {
	author := &fakeByID{m: &model.Member{ID: 5, Nickname: "阿柚", Role: model.RoleMember, Status: model.MemberActive}}
	s := newHollow(t, author, &fakeAudits{})
	post, _ := s.CreatePost(ctxBg, 5, "share", "第一句", nil, true, "")
	// 作者本人没有红点
	if v, _ := s.badges.Read(ctxBg, 5, "hollow"); v {
		t.Fatalf("发帖人自己不应有回应红点")
	}
	// 他人回应 → 作者红点置位
	replier := &fakeByID{m: &model.Member{ID: 9, Nickname: "小满妈妈", Role: model.RoleMember, Status: model.MemberActive}}
	s2 := NewHollowService(s.posts, s.floors, s.react, replier, &fakeAudits{}, audit.MockAuditor{}, ratelimit.NewMemory(), idempot.NewMemory(), s.badges)
	if _, err := s2.CreateFloor(ctxBg, 9, post.ID, "我先接住你", nil, true, ""); err != nil {
		t.Fatalf("回应: %v", err)
	}
	if v, _ := s.badges.Read(ctxBg, 5, "hollow"); !v {
		t.Fatalf("回应后作者应有 hollow 红点")
	}
}

func strPtr(s string) *string { return &s }
