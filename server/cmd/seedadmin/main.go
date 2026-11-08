// seedadmin 创建管理员成员（每个环境一次性工具）。
// 用法（在 server/ 目录，需 DB_DSN 环境）：
//
//	go run ./cmd/seedadmin -phone 13800000001 -name 主管理员
//
// 与正式准入同构：手机号 AES-GCM 加密入 member_secure、HMAC 摘要查找；
// 重复执行幂等退出，不重复创建。生产必须先设置 DB_DSN 与 FILE_KEY。
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/zhumin2/wumingtree-app/server/internal/config"
	"github.com/zhumin2/wumingtree-app/server/internal/model"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/crypto"
	"github.com/zhumin2/wumingtree-app/server/internal/repository"
	"github.com/zhumin2/wumingtree-app/server/internal/service"
)

func main() {
	phone := flag.String("phone", "", "管理员手机号（11 位）")
	name := flag.String("name", "主管理员", "称呼（1-8 字）")
	flag.Parse()

	if *phone == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "用法: seedadmin -phone 138... -name 主管理员")
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("配置加载失败", "err", err)
		os.Exit(1)
	}
	if cfg.DBDSN == "" {
		slog.Error("需要 DB_DSN 环境变量")
		os.Exit(1)
	}
	key, err := hex.DecodeString(cfg.FileKey)
	if err != nil || len(key) != 32 {
		slog.Error("FILE_KEY 非法（64 位 hex）", "err", err)
		os.Exit(1)
	}
	phoneCrypto, err := crypto.NewPhoneCrypto(key)
	if err != nil {
		slog.Error("加解密器构造失败", "err", err)
		os.Exit(1)
	}

	nickname, err := service.NormalizeNickname(*name)
	if err != nil {
		slog.Error("称呼非法（1-8 字）", "err", err)
		os.Exit(1)
	}
	normalized, err := crypto.Normalize(*phone)
	if err != nil {
		slog.Error("手机号非法", "err", err)
		os.Exit(1)
	}

	db, err := gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		slog.Error("MySQL 连接失败", "err", err)
		os.Exit(1)
	}

	hash := phoneCrypto.Hash(normalized)
	repo := &repository.MemberRepo{DB: db}
	if existing, _, err := repo.FindSecureByPhoneHash(hash); err == nil && existing != nil {
		slog.Info("管理员已存在，幂等退出", "memberId", existing.ID, "role", existing.Role)
		return
	}

	cipher, err := phoneCrypto.Encrypt(normalized)
	if err != nil {
		slog.Error("加密失败", "err", err)
		os.Exit(1)
	}
	m := &model.Member{
		Nickname:     nickname,
		Role:         model.RoleAdmin,
		Status:       model.MemberActive,
		DefaultAnon:  true,
		DND:          true,
		NotifyNotice: true,
		NotifyHollow: true,
	}
	sec := &model.MemberSecure{PhoneCipher: cipher, PhoneHash: hash}
	if err := repo.CreateWithSecure(m, sec); err != nil {
		slog.Error("创建失败", "err", err)
		os.Exit(1)
	}
	slog.Info("管理员已创建", "memberId", m.ID, "nickname", nickname)
	fmt.Println(m.ID)
}

var _ = code.CodeOK // 保持 code 包引用（错误统一出口约定）
