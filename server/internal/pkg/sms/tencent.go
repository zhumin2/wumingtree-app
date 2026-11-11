package sms

import (
	"context"
	"log/slog"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/tc3"
)

// TencentSender 腾讯云短信发送器（SendSms v2021-01-11，Host sms.tencentcloudapi.com）。
// 文档：https://cloud.tencent.com/document/product/382/55981。
type TencentSender struct {
	TC         *tc3.Client
	SdkAppID   string
	SignName   string
	TemplateID string
}

func (s TencentSender) SendVerifyCode(ctx context.Context, phone, verifyCode string) error {
	// 国内号码 E.164：+86 + 11 位手机号（phone 已在上层归一化）。
	payload := map[string]any{
		"PhoneNumberSet":   []string{"+86" + phone},
		"SmsSdkAppId":      s.SdkAppID,
		"SignName":         s.SignName,
		"TemplateId":       s.TemplateID,
		"TemplateParamSet": []string{verifyCode},
	}
	var out struct {
		SendStatusSet []struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"SendStatusSet"`
	}
	if err := s.TC.Do(ctx, "SendSms", "2021-01-11", payload, &out); err != nil {
		slog.Warn("sms(tencent) api error", "err", err, "phone", maskPhone(phone))
		return code.ErrDependency
	}
	if len(out.SendStatusSet) == 0 || out.SendStatusSet[0].Code != "Ok" {
		stCode, stMsg := "", ""
		if len(out.SendStatusSet) > 0 {
			stCode, stMsg = out.SendStatusSet[0].Code, out.SendStatusSet[0].Message
		}
		slog.Warn("sms(tencent) send failed", "code", stCode, "message", stMsg, "phone", maskPhone(phone))
		return code.ErrDependency
	}
	return nil
}
