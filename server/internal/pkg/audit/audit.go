// Package audit 内容审核钩子（架构设计 3.4 模式 3 + 6.1/6.2）。
//
// AUDIT_MODE=mock：本地关键词表（词库由运营维护，此处仅示意最小集）；
// AUDIT_MODE=remote：云内容安全 API（签名模板就绪后接入）。
//
// 铁律（架构设计 6.2）：**危机词分支永远不拦截、不延迟**——命中危机词正常发布，
// 同时置 Crisis=true，由上层叠加求助卡并高优通知管理员。
package audit

import (
	"strings"
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

// 词库（运营维护；remote 模式下由云服务替代）。
var (
	crisisWords  = []string{"自杀", "自残", "自伤", "轻生", "不想活", "伤害孩子", "打孩子", "虐待"}
	rejectWords  = []string{"加微信", "加qq", "加QQ", "代购", "刷单", "博彩", "贷款", "兼职广告"}
	pendingWords = []string{"傻", "滚", "蠢", "废物"}
)

// MockAuditor 关键词表实现。
type MockAuditor struct{}

func (MockAuditor) CheckText(text string) Result {
	t := strings.ToLower(text)
	res := Result{Action: ActionAllow}
	for _, w := range crisisWords {
		if strings.Contains(t, w) {
			// 危机词：不拦截、不延迟（红线），但保留 Crisis 标记供上层叠加求助卡
			res.Crisis = true
			res.Hit = w
			return res // 危机优先级最高：即使同时命中违规词也先放行
		}
	}
	for _, w := range rejectWords {
		if strings.Contains(t, strings.ToLower(w)) {
			return Result{Action: ActionReject, Hit: w}
		}
	}
	for _, w := range pendingWords {
		if strings.Contains(t, w) {
			return Result{Action: ActionPending, Hit: w}
		}
	}
	return res
}

// RemoteAuditor 云内容安全占位：接入前按 allow 放行并记日志（上游必须同时启用人工复核）。
type RemoteAuditor struct {
	Secret string
}

func (RemoteAuditor) CheckText(text string) Result {
	// TODO(M2 长周期): 云内容安全文本 API（同步机审，拒绝级/可疑级映射同 mock）。
	return Result{Action: ActionAllow}
}

// New 按配置选择机审实现。
func New(mode, secret string) Auditor {
	if mode == "remote" {
		return RemoteAuditor{Secret: secret}
	}
	return MockAuditor{}
}
