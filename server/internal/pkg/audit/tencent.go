package audit

import (
	"context"
	"encoding/base64"
	"log/slog"
	"strings"
	"time"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/tc3"
)

// cloudTimeout 同步机审的超时上限：CheckText 无 ctx（接口沿用 M2 口径），
// 以此兜底不拖慢发帖链路。
const cloudTimeout = 5 * time.Second

// TencentAuditor 腾讯云文本内容安全机审
// （TextModeration v2020-12-29，Host tms.tencentcloudapi.com，Region 必填）。
// 文档：https://cloud.tencent.com/document/product/1124/51860。
type TencentAuditor struct {
	TC *tc3.Client
}

func (a TencentAuditor) CheckText(text string) Result {
	// 红线（架构设计 6.2）：危机词本地先行——不外发、不延迟，命中正常放行并叠加密报。
	if w, hit := findHit(text, crisisWords); hit {
		return Result{Action: ActionAllow, Crisis: true, Hit: w}
	}
	// 本地词表先行（零成本；与 mock 同口径，云服务覆盖长尾变体）。
	if w, hit := findHit(text, rejectWords); hit {
		return Result{Action: ActionReject, Hit: w}
	}
	if w, hit := findHit(text, pendingWords); hit {
		return Result{Action: ActionPending, Hit: w}
	}
	return a.checkCloud(text)
}

func (a TencentAuditor) checkCloud(text string) Result {
	ctx, cancel := context.WithTimeout(context.Background(), cloudTimeout)
	defer cancel()

	payload := map[string]any{
		"Content": base64.StdEncoding.EncodeToString([]byte(text)),
	}
	var out struct {
		Suggestion string   `json:"Suggestion"` // Block / Review / Pass
		Label      string   `json:"Label"`      // 如 Abuse / Porn / Polity / Illegal
		Keywords   []string `json:"Keywords"`
	}
	if err := a.TC.Do(ctx, "TextModeration", "2020-12-29", payload, &out); err != nil {
		// 云不可用时降级 pending（先发后审语义，不阻塞用户、不漏审）。
		slog.Error("audit(tencent) 云机审不可用，降级 pending（先发后审）", "err", err)
		return Result{Action: ActionPending}
	}
	switch out.Suggestion {
	case "Block":
		return Result{Action: ActionReject, Hit: cloudHit(out.Label, out.Keywords)}
	case "Pass":
		return Result{Action: ActionAllow}
	default: // Review 及未知值：可疑处理（先发后审），未知值记日志便于排查
		if out.Suggestion != "Review" {
			slog.Warn("audit(tencent) 未知 Suggestion，按 pending 处理", "suggestion", out.Suggestion)
		}
		return Result{Action: ActionPending, Hit: cloudHit(out.Label, out.Keywords)}
	}
}

// cloudHit 复核线索：优先命中关键词，其次标签；仅日志/复核用，不回传客户端。
func cloudHit(label string, keywords []string) string {
	if len(keywords) > 0 {
		return strings.Join(keywords, ",")
	}
	return label
}
