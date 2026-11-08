package service

import (
	"context"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/crypto"
)

// AuditAppender 审计落库。
type AuditAppender interface {
	Append(actorID int64, action, targetType string, targetID int64, ip, ua string) error
}

// AuditWriter 需要审计的敏感读取与导出（架构设计 5.4）。
type AuditWriter struct {
	cipher SecureCipherReader
	phone  *crypto.PhoneCrypto
	audit  AuditAppender
}

func NewAuditWriter(cipher SecureCipherReader, phone *crypto.PhoneCrypto, audit AuditAppender) *AuditWriter {
	return &AuditWriter{cipher: cipher, phone: phone, audit: audit}
}

// PlainPhone 管理台查看手机号明文：唯一解密出口，逐次写 phone.view 审计（架构设计 5.3）。
func (w *AuditWriter) PlainPhone(ctx context.Context, memberID, adminID int64, ip, ua string) (string, error) {
	cipher, err := w.cipher.SecureCipherByMemberID(memberID)
	if err != nil || cipher == "" {
		return "", code.ErrNotFound
	}
	plain, err := w.phone.Decrypt(cipher)
	if err != nil {
		return "", err
	}
	if err := w.audit.Append(adminID, "phone.view", "member", memberID, ip, ua); err != nil {
		return "", err
	}
	return plain, nil
}
