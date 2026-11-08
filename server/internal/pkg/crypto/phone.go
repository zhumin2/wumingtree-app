// Package crypto 手机号隔离存储的加解密（架构设计 3.3）。
//
// 密文：AES-256-GCM（随机 12 字节 nonce 前置于密文），主密钥来自部署 env 的 FILE_KEY；
// 摘要：HMAC-SHA256（域分隔前缀），用于登录查找与防重复申请，不可逆。
// member 主表与任何普通接口不出现明文手机号；明文仅管理台独立端点解出并写审计。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrKeyLen     = errors.New("crypto: 主密钥必须为 32 字节（FILE_KEY 64 位 hex）")
	ErrCipherText = errors.New("crypto: 密文无效或被篡改")
	ErrPhone      = errors.New("crypto: 手机号格式不正确")
)

var phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

// PhoneCrypto 手机号加解密器。零值不可用，经 NewPhoneCrypto 构造。
type PhoneCrypto struct {
	key []byte
}

func NewPhoneCrypto(key []byte) (*PhoneCrypto, error) {
	if len(key) != 32 {
		return nil, ErrKeyLen
	}
	cp := make([]byte, 32)
	copy(cp, key)
	return &PhoneCrypto{key: cp}, nil
}

// Normalize 归一手机号：去空格与连字符、去 +86 前缀，校验大陆 11 位号段。
func Normalize(phone string) (string, error) {
	s := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '\t', '　':
			return -1
		}
		return r
	}, strings.TrimSpace(phone))
	s = strings.TrimPrefix(strings.TrimPrefix(s, "+86"), "86")
	if !phoneRe.MatchString(s) {
		return "", ErrPhone
	}
	return s, nil
}

// Encrypt 明文手机号 → 密文（nonce||ciphertext，base64 便于日志展示为占位）。
func (p *PhoneCrypto) Encrypt(phone string) (string, error) {
	block, err := aes.NewCipher(p.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(phone), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 密文 → 明文手机号。篡改或密钥不符时返回 ErrCipherText。
func (p *PhoneCrypto) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrCipherText
	}
	block, err := aes.NewCipher(p.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", ErrCipherText
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", ErrCipherText
	}
	return string(plain), nil
}

// Hash 手机号 → HMAC-SHA256 摘要（hex 64 位）。域分隔，与加密同钥不同用途。
func (p *PhoneCrypto) Hash(phone string) string {
	mac := hmac.New(sha256.New, p.key)
	mac.Write([]byte("phone-hash|" + phone))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// Mask 掩码展示（报名页 / 设置页用）：138****1234。客户端拿不到完整号。
func Mask(phone string) (string, error) {
	if !phoneRe.MatchString(phone) {
		return "", ErrPhone
	}
	return phone[:3] + "****" + phone[7:], nil
}
