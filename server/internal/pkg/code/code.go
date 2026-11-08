// Package code 业务数字错误码与哨兵错误。
//
// 唯一事实来源：《无名树App技术架构设计-v1.0》3.2、《无名树App细化设计-v1.0》第 3 节。
// 约定：
//   - service 层只返回本包哨兵错误（可用 fmt.Errorf("...: %w", ...) 包装上下文）；
//   - module 层经 httpx.Fail 用 errors.As 统一映射，业务代码不得散落数字；
//   - Message 为用户可读文案，禁止携带内部细节。
package code

import "errors"

// 业务错误码。HTTP 恒为 200（网络/网关层错误除外），业务结果看 code。
const (
	CodeOK              int64 = 0
	CodeValidation      int64 = 1001 // 参数或校验失败（data.errors 逐字段）
	CodeUnauthorized    int64 = 2001 // 未登录 / Token 过期
	CodeTokenReuse      int64 = 2002 // 刷新令牌复用，会话已吊销
	CodeSmsCodeInvalid  int64 = 3001 // 验证码错误或过期
	CodeInviteInvalid   int64 = 3002 // 邀请码无效 / 过期 / 次数用尽
	CodeRateLimited     int64 = 4001 // 操作过于频繁
	CodeAuditRejected   int64 = 4101 // 内容审核未通过
	CodeQuotaFull       int64 = 4201 // 名额已满
	CodeDuplicateEnroll int64 = 4202 // 重复报名
	CodePhoneDuplicate  int64 = 4203 // 手机号已有成员或未结申请
	CodeConsentRequired int64 = 4204 // 缺少必要同意
	CodeNotFound        int64 = 5001 // 目标不存在或已删除（越权 / 未报名访问档案统一映射，不泄露存在性）
	CodeNoPermission    int64 = 5002 // 无权限（文件夹锁定 / 家长间私信 / 非作者删帖）
	CodeInternal        int64 = 9000 // 服务内部错误
	CodeDependency      int64 = 9001 // 外部依赖不可用（机审 / 短信降级）
)

// BizError 携带业务码与用户可读文案的哨兵错误。
type BizError struct {
	Code    int64
	Message string
}

func (e *BizError) Error() string { return e.Message }

// 哨兵错误。service 层 errors.Is / %w 使用；新增码先加这里。
var (
	ErrValidation      = &BizError{CodeValidation, "请检查填写的内容"}
	ErrUnauthorized    = &BizError{CodeUnauthorized, "请先登录"}
	ErrTokenReuse      = &BizError{CodeTokenReuse, "登录已过期，请重新登录"}
	ErrSmsCodeInvalid  = &BizError{CodeSmsCodeInvalid, "验证码不正确或已过期"}
	ErrInviteInvalid   = &BizError{CodeInviteInvalid, "邀请码无效，请核对后重试"}
	ErrRateLimited     = &BizError{CodeRateLimited, "操作有点快，过一会再试试"}
	ErrAuditRejected   = &BizError{CodeAuditRejected, "内容未通过检查，请修改后再发"}
	ErrQuotaFull       = &BizError{CodeQuotaFull, "名额已满"}
	ErrDuplicateEnroll = &BizError{CodeDuplicateEnroll, "这个孩子已经在名单里了"}
	ErrPhoneDuplicate  = &BizError{CodePhoneDuplicate, "这个手机号已经申请过，可以直接登录"}
	ErrConsentRequired = &BizError{CodeConsentRequired, "需要勾选两条同意"}
	ErrNotFound        = &BizError{CodeNotFound, "内容不存在或已删除"}
	ErrNoPermission    = &BizError{CodeNoPermission, "暂无权限"}
	ErrInternal        = &BizError{CodeInternal, "服务开小差了，请稍后再试"}
	ErrDependency      = &BizError{CodeDependency, "服务暂时不可用，请稍后再试"}
)

// Of 返回 err 对应的业务码与文案。
// 未知错误统一归 9000 且只返回通用文案，不向客户端泄露内部信息。
func Of(err error) (int64, string) {
	var be *BizError
	if errors.As(err, &be) {
		return be.Code, be.Message
	}
	return CodeInternal, ErrInternal.Message
}
