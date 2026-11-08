package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/audit"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/idempot"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/ratelimit"
)

// ---- 树洞存储接口（消费方定义；GORM 实现在 repository）----

// Page 时间线分页（游标见 pkg/cursor；limit 由服务钳制）。
type Page struct {
	Cursor string
	Limit  int
}

// PostRow 帖子行（扁平字段：gorm 对带主键的嵌套结构会按关联解析，故不嵌套）。
type PostRow struct {
	ID             int64
	AuthorID       int64
	Mood           string
	Title          string
	Body           *string
	IsAnonymous    bool
	Status         string
	RemovedReason  *string
	ReviewStatus   string
	PublishedAt    time.Time
	AuthorNickname string
}

// FloorRow 楼层行（同上扁平）。
type FloorRow struct {
	ID             int64
	PostID         int64
	ParentFloorID  *int64
	Depth          int8
	AuthorID       int64
	Content        string
	IsAnonymous    bool
	Status         string
	CreatedAt      time.Time
	AuthorNickname string
}

// PostStore 帖子存取。
type PostStore interface {
	Create(p *model.Post) error
	FindByID(id int64) (*PostRow, error)
	List(mood string, page Page) ([]PostRow, string, error)
	ListByAuthor(authorID int64, page Page) ([]PostRow, string, error)
	UpdateStatus(id int64, status, removedReason string) error
}

// FloorStore 楼层存取。
type FloorStore interface {
	Create(f *model.Floor) error
	FindByID(id int64) (*model.Floor, error)
	CountPublished(postID int64) (int64, error)
	CountPublishedByPosts(postIDs []int64) (map[int64]int64, error)
	ListByPost(postID int64, page Page) ([]FloorRow, string, error)
}

// ReactionStore 回应存取。
type ReactionStore interface {
	Toggle(memberID int64, targetType string, targetID int64, rtype string) (bool, error)
	Count(targetType string, targetID int64, rtype string) (int64, error)
	Mine(memberID int64, targetType string, targetID int64) ([]string, error)
	MineByTargets(memberID int64, targetType string, ids []int64) (map[int64][]string, error)
	CountByTargets(targetType string, ids []int64) (map[int64]map[string]int64, error)
}

// ReportStore 举报存取。
type ReportStore interface {
	Create(rep *model.Report) error
	ListByStatus(status string, offset, limit int) ([]model.Report, int64, error)
	Handle(reportID, adminID int64, action, note string) error
}

// BadgeStore 红点位标记（仅布尔，无聚合计数——红线）。
type BadgeStore interface {
	Mark(ctx context.Context, memberID int64, kind string) error
	Clear(ctx context.Context, memberID int64, kind string) error
	Read(ctx context.Context, memberID int64, kind string) (bool, error)
}

// ---- VO ----

// PostVO 树洞帖视图。红线：匿名帖物理上不含 authorId / 昵称字段。
type PostVO struct {
	ID          int64     `json:"id"`
	Mood        string    `json:"mood"`
	Title       string    `json:"title"`
	Body        string    `json:"body,omitempty"`
	IsAnonymous bool      `json:"isAnonymous"`
	DisplayName string    `json:"displayName"`
	Replies     int64     `json:"replies"`
	Hug         int64     `json:"hug"`
	MeToo       int64     `json:"meToo"`
	MyReactions []string  `json:"myReactions"`
	PublishedAt time.Time `json:"publishedAt"`
}

// PostPageVO 树洞列表分页。
type PostPageVO struct {
	Items      []PostVO `json:"items"`
	NextCursor *string  `json:"nextCursor"`
}

// FloorVO 楼层视图（depth 1 或 2）。
type FloorVO struct {
	ID          int64     `json:"id"`
	ParentID    *int64    `json:"parentFloorId"`
	Depth       int       `json:"depth"`
	DisplayName string    `json:"displayName"`
	Content     string    `json:"content"`
	Hug         int64     `json:"hug"`
	Helpful     int64     `json:"helpful"`
	CreatedAt   time.Time `json:"createdAt"`
}

// PostDetailVO 帖子详情 + 首屏楼层。Crisis=true 时客户端立即叠加求助卡（红线）。
type PostDetailVO struct {
	Post   PostVO    `json:"post"`
	Floors []FloorVO `json:"floors"`
	Crisis bool      `json:"crisis"`
}

// ReactionStateVO 回应 toggle 结果。
type ReactionStateVO struct {
	Active bool  `json:"active"`
	Count  int64 `json:"count"`
}

// ---- 常量与工具 ----

const idemTTL = 24 * time.Hour

func idemKey(kind, token string) string { return "idem:" + kind + ":" + token }

func int64Key(v int64) string { return strconv.FormatInt(v, 10) }

func bodyText(b *string) string {
	if b == nil {
		return ""
	}
	return *b
}

func validMood(m string) bool {
	switch m {
	case "share", "tired", "ask", "talk":
		return true
	}
	return false
}

// ---- HollowService ----

// HollowService 树洞：发帖 / 楼层 / 回应 / 我的树洞。
type HollowService struct {
	posts   PostStore
	floors  FloorStore
	react   ReactionStore
	members MemberByID
	audits  AuditAppender
	audit   audit.Auditor
	limiter ratelimit.Limiter
	idem    idempot.Store
	badges  BadgeStore
	Now     func() time.Time
}

func NewHollowService(
	posts PostStore, floors FloorStore, react ReactionStore,
	members MemberByID, audits AuditAppender, auditor audit.Auditor,
	limiter ratelimit.Limiter, idem idempot.Store, badges BadgeStore,
) *HollowService {
	return &HollowService{
		posts: posts, floors: floors, react: react, members: members,
		audits: audits, audit: auditor, limiter: limiter, idem: idem,
		badges: badges, Now: time.Now,
	}
}

// CreatePost 发帖。顺序固定：校验 → 幂等（先于频控——铁律）→ 频控 → 机审 → 入库。
// 危机词分支不拦截：正常发布 + 危机标记 + 高优告警（架构设计 6.2）。
func (s *HollowService) CreatePost(ctx context.Context, memberID int64, mood, title string, body *string, isAnon bool, clientToken string) (*PostVO, error) {
	if !validMood(mood) {
		return nil, code.ErrValidation
	}
	t := trimSpaces(title)
	if n := len([]rune(t)); n < 1 || n > 50 {
		return nil, code.ErrValidation
	}
	if body != nil && len([]rune(*body)) > 2000 {
		return nil, code.ErrValidation
	}

	// 1) 幂等（铁律：先于频控）
	var key string
	if clientToken != "" {
		key = idemKey("post", clientToken)
		first, err := s.idem.Acquire(ctx, key, idemTTL)
		if err != nil {
			return nil, err
		}
		if !first {
			return s.loadIdempotentPost(ctx, key)
		}
	}

	// 2-4) 频控 → 机审 → 入库（业务失败须释放槽位，允许客户端重试）
	vo, execErr := s.execCreatePost(ctx, memberID, mood, t, body, isAnon)
	if execErr != nil {
		if key != "" {
			_ = s.idem.Release(ctx, key)
		}
		return nil, execErr
	}
	if clientToken != "" {
		if payload, jerr := json.Marshal(vo); jerr == nil {
			_ = s.idem.Save(ctx, key, payload, idemTTL)
		}
	}
	return vo, nil
}

// loadIdempotentPost 幂等命中：返回首次结果；首次仍在执行中则提示稍后再试。
func (s *HollowService) loadIdempotentPost(ctx context.Context, key string) (*PostVO, error) {
	payload, ok, err := s.idem.Load(ctx, key)
	if err != nil {
		return nil, err
	}
	if ok && len(payload) > 0 {
		var vo PostVO
		if json.Unmarshal(payload, &vo) == nil {
			return &vo, nil
		}
	}
	return nil, code.ErrRateLimited
}

// execCreatePost 频控 → 机审 → 入库 → 危机告警。
func (s *HollowService) execCreatePost(ctx context.Context, memberID int64, mood, title string, body *string, isAnon bool) (*PostVO, error) {
	// 频控：10 条/天（细化设计 6）
	if ok, _ := s.limiter.Allow(ctx, "post:day:"+int64Key(memberID), 10, 24*time.Hour); !ok {
		return nil, code.ErrRateLimited
	}
	res := s.audit.CheckText(title + "\n" + bodyText(body))
	if res.Action == audit.ActionReject {
		return nil, code.ErrAuditRejected
	}
	reviewStatus := model.ReviewClean
	if res.Action == audit.ActionPending {
		reviewStatus = model.ReviewPending
	}
	p := model.Post{
		AuthorID:     memberID,
		Mood:         mood,
		Title:        title,
		Body:         body,
		IsAnonymous:  isAnon,
		Status:       model.PostPublished,
		ReviewStatus: reviewStatus,
		PublishedAt:  s.Now(),
	}
	if err := s.posts.Create(&p); err != nil {
		return nil, err
	}
	if res.Crisis {
		s.alertCrisis(ctx, memberID, "post", p.ID, res.Hit)
	}
	row := PostRow{
		ID: p.ID, AuthorID: p.AuthorID, Mood: p.Mood, Title: p.Title,
		Body: p.Body, IsAnonymous: p.IsAnonymous, Status: p.Status,
		ReviewStatus: p.ReviewStatus, PublishedAt: p.PublishedAt,
		AuthorNickname: s.nicknameOf(memberID),
	}
	return s.postVO(&row, 0, nil), nil
}

// execCreateFloor 频控 → 楼层校验 → 机审 → 入库 → 红点/告警。
func (s *HollowService) execCreateFloor(ctx context.Context, memberID, postID int64, content string, parentFloorID *int64, isAnon bool) (*FloorVO, error) {
	post, err := s.posts.FindByID(postID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status != model.PostPublished {
		return nil, code.ErrNotFound
	}
	if ok, _ := s.limiter.Allow(ctx, "floor:day:"+int64Key(memberID), 50, 24*time.Hour); !ok {
		return nil, code.ErrRateLimited
	}
	depth := int8(1)
	if parentFloorID != nil {
		parent, err := s.floors.FindByID(*parentFloorID)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.PostID != postID || parent.Depth != 1 || parent.Status != model.FloorPublished {
			return nil, code.ErrValidation
		}
		depth = 2
	}
	res := s.audit.CheckText(content)
	if res.Action == audit.ActionReject {
		return nil, code.ErrAuditRejected
	}
	reviewStatus := model.ReviewClean
	if res.Action == audit.ActionPending {
		reviewStatus = model.ReviewPending
	}
	f := model.Floor{
		PostID:        postID,
		ParentFloorID: parentFloorID,
		Depth:         depth,
		AuthorID:      memberID,
		Content:       content,
		IsAnonymous:   isAnon,
		Status:        model.FloorPublished,
		ReviewStatus:  reviewStatus,
	}
	if err := s.floors.Create(&f); err != nil {
		return nil, err
	}
	if res.Crisis {
		s.alertCrisis(ctx, memberID, "floor", f.ID, res.Hit)
	}
	_ = s.badges.Mark(ctx, post.AuthorID, "hollow")
	row := FloorRow{
		ID:          f.ID,
		PostID:      f.PostID,
		Depth:       f.Depth,
		AuthorID:    f.AuthorID,
		Content:     f.Content,
		IsAnonymous: f.IsAnonymous,
		Status:      f.Status,
		CreatedAt:   f.CreatedAt,
	}
	if !f.IsAnonymous {
		row.AuthorNickname = s.nicknameOf(memberID)
	}
	vo := floorVO(&row)
	return &vo, nil
}

// alertCrisis 危机内容告警：审计留痕 + 高优日志（值班推送 M5 接入）。
func (s *HollowService) alertCrisis(ctx context.Context, memberID int64, targetType string, targetID int64, hit string) {
	_ = s.audits.Append(memberID, "crisis.alert", targetType, targetID, "", "")
	slog.Warn("危机内容高优告警", "memberId", memberID, "type", targetType, "targetId", targetID, "hit", hit)
}

// nicknameOf 作者昵称（作者软删时返回占位）。
func (s *HollowService) nicknameOf(memberID int64) string {
	m, err := s.members.FindByID(memberID)
	if err != nil || m == nil {
		return "已注销成员"
	}
	return m.Nickname
}

// ListPosts 树洞列表：心情筛选 + 时间倒序（红线：无热度排序）。
func (s *HollowService) ListPosts(ctx context.Context, memberID int64, mood string, page Page) (*PostPageVO, error) {
	if mood != "" && !validMood(mood) {
		return nil, code.ErrValidation
	}
	if page.Limit <= 0 || page.Limit > 50 {
		page.Limit = 20
	}
	rows, next, err := s.posts.List(mood, page)
	if err != nil {
		return nil, err
	}
	return s.postPage(ctx, rows, next, memberID)
}

// GetPost 详情 + 首屏楼层；危机标记读时即时推导（审核逻辑单一来源）。
func (s *HollowService) GetPost(ctx context.Context, memberID, postID int64) (*PostDetailVO, error) {
	row, err := s.posts.FindByID(postID)
	if err != nil {
		return nil, err
	}
	if row == nil || row.Status != model.PostPublished {
		return nil, code.ErrNotFound
	}
	vo := s.postVO(row, 0, nil)
	floors, _, err := s.floorVOs(ctx, postID, Page{Limit: 20})
	if err != nil {
		return nil, err
	}
	crisis := s.audit.CheckText(row.Title + "\n" + bodyText(row.Body)).Crisis
	return &PostDetailVO{Post: *vo, Floors: floors, Crisis: crisis}, nil
}

// ListFloors 楼层分页（按时间从上到下）。
func (s *HollowService) ListFloors(ctx context.Context, memberID, postID int64, page Page) ([]FloorVO, *string, error) {
	if page.Limit <= 0 || page.Limit > 100 {
		page.Limit = 20
	}
	if row, err := s.posts.FindByID(postID); err != nil {
		return nil, nil, err
	} else if row == nil || row.Status != model.PostPublished {
		return nil, nil, code.ErrNotFound
	}
	floors, next, err := s.floorVOs(ctx, postID, page)
	if err != nil {
		return nil, nil, err
	}
	if next == "" {
		return floors, nil, nil
	}
	return floors, &next, nil
}

// CreateFloor 发表楼层（可对楼层回应，楼中楼仅一层：depth ≤ 2，CHECK 兜底）。
func (s *HollowService) CreateFloor(ctx context.Context, memberID, postID int64, content string, parentFloorID *int64, isAnon bool, clientToken string) (*FloorVO, error) {
	c := trimSpaces(content)
	if n := len([]rune(c)); n < 1 || n > 500 {
		return nil, code.ErrValidation
	}

	post, err := s.posts.FindByID(postID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status != model.PostPublished {
		return nil, code.ErrNotFound
	}

	// 幂等（先于频控——铁律）
	if clientToken != "" {
		key := idemKey("floor", clientToken)
		first, err := s.idem.Acquire(ctx, key, idemTTL)
		if err != nil {
			return nil, err
		}
		if !first {
			// 楼层结果不缓存计数；首次已在执行或已成功，重试提示稍后即可
			return nil, code.ErrRateLimited
		}
		vo, execErr := s.execCreateFloor(ctx, memberID, postID, c, parentFloorID, isAnon)
		if execErr != nil {
			_ = s.idem.Release(ctx, key)
			return nil, execErr
		}
		return vo, nil
	}
	return s.execCreateFloor(ctx, memberID, postID, c, parentFloorID, isAnon)
}

// TogglePostReaction 帖子级回应 toggle（hug 抱抱 / me_too 我也是）。
func (s *HollowService) TogglePostReaction(ctx context.Context, memberID, postID int64, rtype string) (*ReactionStateVO, error) {
	if rtype != "hug" && rtype != "me_too" {
		return nil, code.ErrValidation
	}
	row, err := s.posts.FindByID(postID)
	if err != nil {
		return nil, err
	}
	if row == nil || row.Status != model.PostPublished {
		return nil, code.ErrNotFound
	}
	active, err := s.react.Toggle(memberID, "post", postID, rtype)
	if err != nil {
		return nil, err
	}
	n, err := s.react.Count("post", postID, rtype)
	if err != nil {
		return nil, err
	}
	return &ReactionStateVO{Active: active, Count: n}, nil
}

// ToggleFloorReaction 楼层级回应 toggle（hug 抱一下 / helpful 有帮助）。
func (s *HollowService) ToggleFloorReaction(ctx context.Context, memberID, floorID int64, rtype string) (*ReactionStateVO, error) {
	if rtype != "hug" && rtype != "helpful" {
		return nil, code.ErrValidation
	}
	f, err := s.floors.FindByID(floorID)
	if err != nil {
		return nil, err
	}
	if f == nil || f.Status != model.FloorPublished {
		return nil, code.ErrNotFound
	}
	if post, err := s.posts.FindByID(f.PostID); err != nil {
		return nil, err
	} else if post == nil || post.Status != model.PostPublished {
		return nil, code.ErrNotFound
	}
	active, err := s.react.Toggle(memberID, "floor", floorID, rtype)
	if err != nil {
		return nil, err
	}
	n, err := s.react.Count("floor", floorID, rtype)
	if err != nil {
		return nil, err
	}
	return &ReactionStateVO{Active: active, Count: n}, nil
}

// DeletePost 删帖（作者或管理员）；楼层随帖子整体隐藏（软删，不逐条删除）。
func (s *HollowService) DeletePost(ctx context.Context, memberID, postID int64) error {
	row, err := s.posts.FindByID(postID)
	if err != nil {
		return err
	}
	if row == nil || row.Status != model.PostPublished {
		return code.ErrNotFound
	}
	m, err := s.members.FindByID(memberID)
	if err != nil {
		return err
	}
	if row.AuthorID != memberID && (m == nil || m.Role != model.RoleAdmin) {
		return code.ErrNoPermission
	}
	reason := "by_author"
	if row.AuthorID != memberID {
		reason = "by_report" // 管理员处置复用该枚举（DDL 注释三种来源之一）
	}
	return s.posts.UpdateStatus(postID, model.PostRemoved, reason)
}

// MyPosts 我的树洞（含匿名帖；服务端按作者过滤——红线）。
// 产品规则：匿名帖也只在这里显示为自己的记录（对外列表仍显示「一位家长」）。
func (s *HollowService) MyPosts(ctx context.Context, memberID int64, page Page) (*PostPageVO, error) {
	if page.Limit <= 0 || page.Limit > 50 {
		page.Limit = 20
	}
	rows, next, err := s.posts.ListByAuthor(memberID, page)
	if err != nil {
		return nil, err
	}
	pageVO, err := s.postPage(ctx, rows, next, memberID)
	if err != nil {
		return nil, err
	}
	mine := s.nicknameOf(memberID)
	for i := range pageVO.Items {
		pageVO.Items[i].DisplayName = mine
	}
	return pageVO, nil
}

// ---- 内部组装 ----

// postVO 单帖视图组装（红线：匿名帖不含 authorId 与昵称，displayName 按匿名解析）。
func (s *HollowService) postVO(row *PostRow, replies int64, my []string) *PostVO {
	display := "一位家长"
	if !row.IsAnonymous {
		display = row.AuthorNickname
	}
	vo := &PostVO{
		ID:          row.ID,
		Mood:        row.Mood,
		Title:       row.Title,
		IsAnonymous: row.IsAnonymous,
		DisplayName: display,
		Replies:     replies,
		MyReactions: my,
		PublishedAt: row.PublishedAt,
	}
	if row.Body != nil {
		vo.Body = *row.Body
	}
	return vo
}

// postPage 列表组装：批量取回应计数 / 我的回应 / 回应数（避免 N+1 逐条查询）。
func (s *HollowService) postPage(ctx context.Context, rows []PostRow, next string, viewerID int64) (*PostPageVO, error) {
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	replyCounts, err := s.floors.CountPublishedByPosts(ids)
	if err != nil {
		return nil, err
	}
	reactCounts, err := s.react.CountByTargets("post", ids)
	if err != nil {
		return nil, err
	}
	mine, err := s.react.MineByTargets(viewerID, "post", ids)
	if err != nil {
		return nil, err
	}
	items := make([]PostVO, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		vo := s.postVO(r, replyCounts[r.ID], mine[r.ID])
		vo.Hug = reactCounts[r.ID]["hug"]
		vo.MeToo = reactCounts[r.ID]["me_too"]
		items = append(items, *vo)
	}
	page := &PostPageVO{Items: items}
	if next != "" {
		page.NextCursor = &next
	}
	return page, nil
}

// floorVO 单楼层视图（匿名显示名解析——红线）。
func floorVO(row *FloorRow) FloorVO {
	display := "一位家长"
	if !row.IsAnonymous {
		display = row.AuthorNickname
	}
	return FloorVO{
		ID:          row.ID,
		ParentID:    row.ParentFloorID,
		Depth:       int(row.Depth),
		DisplayName: display,
		Content:     row.Content,
		CreatedAt:   row.CreatedAt,
	}
}

// floorVOs 楼层视图组装（批量计数）。
func (s *HollowService) floorVOs(ctx context.Context, postID int64, page Page) ([]FloorVO, string, error) {
	rows, next, err := s.floors.ListByPost(postID, page)
	if err != nil {
		return nil, "", err
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	counts, err := s.react.CountByTargets("floor", ids)
	if err != nil {
		return nil, "", err
	}
	out := make([]FloorVO, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		vo := floorVO(r)
		vo.Hug = counts[r.ID]["hug"]
		vo.Helpful = counts[r.ID]["helpful"]
		out = append(out, vo)
	}
	return out, next, nil
}
