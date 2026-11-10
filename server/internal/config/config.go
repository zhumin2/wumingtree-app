// Package config 服务配置：全部来自环境变量。
// 生产由 systemd EnvironmentFile（/opt/wumingtree/wumingtree.env）注入；
// 密钥不入仓库（架构设计 5.1）。
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// Config 运行配置。
type Config struct {
	Env           string // dev / staging / prod
	Addr          string // HTTP 监听地址，默认 :8080
	DBDSN         string // MySQL DSN；为空时跳过连接（healthz 报告 db=false）
	RedisAddr     string // 为空时跳过连接
	RedisPassword string
	JWTSecret     string
	FileKey       string // 手机号加密主密钥：64 位 hex（=32 字节 AES-256 key）
	SMSType       string // mock / remote（mock 固定验证码，联调用）
	SMSSecret     string // remote 模式的云厂商凭证
	AuditMode     string // mock / remote（mock 关键词表）
	AuditSecret   string // remote 模式的云内容安全凭证
	PublicBaseURL string // 文件分发外链基地址（架构设计 3.9 的抽象）
	UploadsDir    string // 本地上传根目录，默认 /opt/wumingtree/uploads
	WebDir        string // Flutter Web 用户端产物目录；为空则不托管（架构设计 5.2）
}

// Load 读取配置并做最小校验。prod 缺密钥直接启动失败（宁可起不来，不可裸奔）。
func Load() (Config, error) {
	c := Config{
		Env:           getEnv("ENV", "dev"),
		Addr:          getEnv("APP_ADDR", ":8080"),
		DBDSN:         strings.TrimSpace(os.Getenv("DB_DSN")),
		RedisAddr:     strings.TrimSpace(os.Getenv("REDIS_ADDR")),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		FileKey:       strings.TrimSpace(os.Getenv("FILE_KEY")),
		SMSType:       getEnv("SMS_TYPE", "mock"),
		SMSSecret:     os.Getenv("SMS_SECRET"),
		AuditMode:     getEnv("AUDIT_MODE", "mock"),
		AuditSecret:   os.Getenv("AUDIT_SECRET"),
		PublicBaseURL: strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/"),
		UploadsDir:    getEnv("UPLOADS_DIR", "/opt/wumingtree/uploads"),
		WebDir:        strings.TrimSpace(os.Getenv("WEB_DIR")),
	}

	if c.JWTSecret == "" {
		if c.Env == "prod" {
			return c, fmt.Errorf("config: prod 环境必须设置 JWT_SECRET")
		}
		c.JWTSecret = "dev-only-secret-not-for-production"
	}
	if c.FileKey == "" {
		if c.Env == "prod" {
			return c, fmt.Errorf("config: prod 环境必须设置 FILE_KEY")
		}
		// dev 固定派生键：保证本地重启后已存密文仍可解（仅限非 prod）
		c.FileKey = hex.EncodeToString(devDerivedKey())
	}
	if key, err := hex.DecodeString(c.FileKey); err != nil || len(key) != 32 {
		return c, fmt.Errorf("config: FILE_KEY 必须是 64 位 hex（32 字节），openssl rand -hex 32 生成")
	}
	switch c.SMSType {
	case "mock", "remote":
	default:
		return c, fmt.Errorf("config: SMS_TYPE 仅支持 mock / remote")
	}
	switch c.AuditMode {
	case "mock", "remote":
	default:
		return c, fmt.Errorf("config: AUDIT_MODE 仅支持 mock / remote")
	}
	return c, nil
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

// devDerivedKey 非生产环境的固定派生密钥（32 字节）。
// 禁止用于生产：prod 必须显式设置 FILE_KEY。
func devDerivedKey() []byte {
	sum := sha256.Sum256([]byte("wumingtree-dev-file-key(not-for-prod)"))
	return sum[:]
}
