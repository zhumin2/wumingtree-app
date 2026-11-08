// Package sms 短信发送（架构设计 3.3）。
// mock：固定验证码便于联调（细化设计：SMS_TYPE=mock）；remote：云厂商 SDK，
// 签名与模板备案就绪后接入（M1 长周期事项），未接入时返回依赖不可用。
package sms

import (
	"context"
	"log/slog"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
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

// RemoteSender 云厂商发送器占位：签名模板就绪后替换实现。
type RemoteSender struct {
	Secret string
}

func (RemoteSender) SendVerifyCode(ctx context.Context, phone, verifyCode string) error {
	// TODO(M1 长周期事项): 接入云厂商短信 SDK（签名 / 模板备案后）。
	return code.ErrDependency
}

// New 按配置选择发送器。
func New(mock bool, secret string) Sender {
	if mock {
		return MockSender{}
	}
	return RemoteSender{Secret: secret}
}
