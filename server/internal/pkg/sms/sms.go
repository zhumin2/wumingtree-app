// Package sms 短信发送（架构设计 3.3）。
// mock：固定验证码便于联调（细化设计：SMS_TYPE=mock）；remote：腾讯云短信
// SendSms v2021-01-11（签名/模板备案完成后启用，见 deploy/linux/wumingtree.env.example）。
package sms

import (
	"context"
	"log/slog"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/tc3"
)

// Sender 发送短信验证码。phone 为归一化后的 11 位号码。
type Sender interface {
	SendVerifyCode(ctx context.Context, phone, verifyCode string) error
}

// MockVerifyCode mock 模式固定验证码（仅 dev / 联调）。
const MockVerifyCode = "123456"

// MockSender mock 发送器：不外发，记录日志即可（联调时从日志取码）。
type MockSender struct{}

func (MockSender) SendVerifyCode(ctx context.Context, phone, verifyCode string) error {
	slog.Info("sms(mock)", "phone", phone, "code", verifyCode)
	return nil
}

// Options remote 模式的腾讯云短信参数（凭证在 TC 内，应用/签名/模板由控制台创建）。
type Options struct {
	TC         *tc3.Client
	SdkAppID   string // 短信控制台 SdkAppId，如 1400006666
	SignName   string // 已备案签名内容，国内短信必填
	TemplateID string // 已备案模板 ID，验证码模板仅收 0-6 位数字参数
}

// New 按配置选择发送器。
func New(mock bool, opts Options) Sender {
	if mock {
		return MockSender{}
	}
	return TencentSender{TC: opts.TC, SdkAppID: opts.SdkAppID, SignName: opts.SignName, TemplateID: opts.TemplateID}
}

// maskPhone 日志脱敏：只留后四位。
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "****" + phone[len(phone)-4:]
}
