// Package audit 内容审核钩子（架构设计 3.4 模式 3 + 6.1/6.2）。
//
// AUDIT_MODE=mock：本地关键词表（词库由运营维护，此处仅示意最小集）；
// AUDIT_MODE=remote：腾讯云文本内容安全 TextModeration（tencent.go），
// 本地危机词/词表仍先行，云服务覆盖长尾。
//
// 铁律（架构设计 6.2）：**危机词分支永远不拦截、不延迟**——命中危机词正常发布，
// 同时置 Crisis=true，由上层叠加求助卡并高优通知管理员。
package audit

import (
	"strings"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/tc3"
)

// Action 机审结论。
const (
	ActionAllow   = "allow"   // 直接发布
	ActionReject  = "reject"  // 拒绝级：不入库，返回 4101
	ActionPending = "pending" // 可疑级：先发后审（review_status=pending）
)

// Result 单次机审结果。
type Result struct {
	Action string
	Crisis bool   // 命中危机词（与 Action 并存：危机不拦截）
	Hit    string // 命中的词（仅日志/复核用，不回传客户端）
}

// Auditor 文本机审接口。
type Auditor interface {
	CheckText(text string) Result
}

// 词库（运营维护；remote 模式下危机词仍本地先行——红线要求不外发不延迟，
// reject/pending 词表作为离线双保险，云服务覆盖长尾）。
var (
	crisisWords  = []string{"自杀", "自残", "自伤", "轻生", "不想活", "伤害孩子", "打孩子", "虐待"}
	rejectWords  = []string{"加微信", "加qq", "加QQ", "代购", "刷单", "博彩", "贷款", "兼职广告"}
	pendingWords = []string{"傻", "滚", "蠢", "废物"}
)

// findHit 返回 text 中命中的第一个词。
func findHit(text string, words []string) (string, bool) {
	t := strings.ToLower(text)
	for _, w := range words {
		if strings.Contains(t, strings.ToLower(w)) {
			return w, true
		}
	}
	return "", false
}

// MockAuditor 关键词表实现。
type MockAuditor struct{}

func (MockAuditor) CheckText(text string) Result {
	res := Result{Action: ActionAllow}
	if w, hit := findHit(text, crisisWords); hit {
		// 危机词：不拦截、不延迟（红线），但保留 Crisis 标记供上层叠加求助卡
		res.Crisis = true
		res.Hit = w
		return res // 危机优先级最高：即使同时命中违规词也先放行
	}
	if w, hit := findHit(text, rejectWords); hit {
		return Result{Action: ActionReject, Hit: w}
	}
	if w, hit := findHit(text, pendingWords); hit {
		return Result{Action: ActionPending, Hit: w}
	}
	return res
}

// Options remote 模式的云内容安全参数（凭证在 TC 内）。
type Options struct {
	TC *tc3.Client
}

// New 按配置选择机审实现。
func New(mode string, opts Options) Auditor {
	if mode == "remote" {
		return TencentAuditor{TC: opts.TC}
	}
	return MockAuditor{}
}
