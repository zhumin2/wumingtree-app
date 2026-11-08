//go:build integration

// 集成测试：真实 MySQL 上应用迁移，走「邀请码申请 → 管理员审核 → 短信登录 → 我的资料」
// 全链路（开发计划 M1-14；CI 由 MySQL service 容器提供数据库）。
// 运行：WMT_IT_DSN="root:root@tcp(127.0.0.1:3306)/wumingtree?parseTime=true&loc=UTC&multiStatements=true" \
//        go test -tags integration ./internal/it/ -v
// 本地无 DB 时自动跳过。
package it

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/zhumin2/wumingtree-app/server/internal/config"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/crypto"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/sms"
	"github.com/zhumin2/wumingtree-app/server/internal/router"
)

const (
	inviteA   = "WMTTESTAAAA01" // 单次有效：主流程用
	inviteB   = "WMTTESTBBBB02" // 多次有效（max 3）：防重/边界用
	adminPhon = "13800000001"
	memberPh  = "13800001234"
)

func itDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("WMT_IT_DSN")
	if dsn == "" {
		t.Skip("未设置 WMT_IT_DSN，跳过集成测试")
	}
	return dsn
}

func applyMigrations(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}
	// down + up 成对重放，保证幂等
	for _, f := range []string{"../../migrations/000001_init.down.sql", "../../migrations/000001_init.up.sql"} {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("读取迁移 %s: %v", f, err)
		}
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			t.Fatalf("执行迁移 %s: %v", f, err)
		}
	}
	return db
}

type itEnv struct {
	e     *gin.Engine
	db    *gorm.DB
	phone *crypto.PhoneCrypto
}

func setup(t *testing.T) *itEnv {
	t.Helper()
	dsn := itDSN(t)
	db := applyMigrations(t, dsn)

	t.Setenv("ENV", "dev")
	t.Setenv("JWT_SECRET", "it-secret-not-for-prod")
	t.Setenv("FILE_KEY", strings.Repeat("ab", 32)) // 64 hex = 32 字节
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	key, err := hex.DecodeString(cfg.FileKey)
	if err != nil {
		t.Fatalf("FILE_KEY: %v", err)
	}
	phoneCrypto, err := crypto.NewPhoneCrypto(key)
	if err != nil {
		t.Fatalf("crypto: %v", err)
	}
	return &itEnv{e: router.New(cfg, db, nil), db: db, phone: phoneCrypto}
}

// call 发请求并解析统一信封（HTTP 恒 200，业务看 code）。
func call(e *gin.Engine, method, path, token string, body any) (int, map[string]any) {
	var buf *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewReader(b)
	} else {
		buf = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	out := map[string]any{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func codeNum(env map[string]any) int64 {
	v, _ := env["code"].(float64)
	return int64(v)
}

func dataMap(env map[string]any) map[string]any {
	d, _ := env["data"].(map[string]any)
	return d
}

func (s *itEnv) seedAdmin(t *testing.T) string {
	t.Helper()
	cipher, err := s.phone.Encrypt(adminPhon)
	if err != nil {
		t.Fatalf("加密: %v", err)
	}
	hash := s.phone.Hash(adminPhon)
	seed := fmt.Sprintf(
		`INSERT INTO member (nickname, role, status, default_anon, dnd, notify_notice, notify_hollow, joined_at, created_at)
		 VALUES ('主管理员', 'admin', 'active', 1, 1, 1, 1, NOW(3), NOW(3));
		 SET @m = LAST_INSERT_ID();
		 INSERT INTO member_secure (member_id, phone_cipher, phone_hash) VALUES (@m, '%s', '%s');`, cipher, hash)
	if err := s.db.Exec(seed).Error; err != nil {
		t.Fatalf("种子管理员: %v", err)
	}
	return s.login(t, adminPhon)
}

func (s *itEnv) seedInvite(t *testing.T, code string, maxUses int) {
	t.Helper()
	err := s.db.Exec(fmt.Sprintf(
		`INSERT INTO invite_code (code, status, valid_from, valid_until, max_uses, used_count, created_by, created_at)
		 VALUES ('%s', 'active', NOW(3) - INTERVAL 1 DAY, NOW(3) + INTERVAL 30 DAY, %d, 0, 1, NOW(3))`, code, maxUses),
	).Error
	if err != nil {
		t.Fatalf("种子邀请码: %v", err)
	}
}

func (s *itEnv) login(t *testing.T, phone string) string {
	t.Helper()
	_, env := call(s.e, "POST", "/api/v1/auth/sms-codes", "", map[string]any{"phone": phone})
	if codeNum(env) != 0 {
		t.Fatalf("发送验证码失败: %v", env)
	}
	_, env = call(s.e, "POST", "/api/v1/auth/login", "", map[string]any{"phone": phone, "code": sms.MockVerifyCode})
	if codeNum(env) != 0 {
		t.Fatalf("登录失败: %v", env)
	}
	toks := dataMap(env)["tokens"].(map[string]any)
	return toks["accessToken"].(string)
}

func TestOnboardingFlow(t *testing.T) {
	s := setup(t)
	adminToken := s.seedAdmin(t)
	s.seedInvite(t, inviteA, 1)
	s.seedInvite(t, inviteB, 3)

	t.Run("未登录访问 admin 被拦截", func(t *testing.T) {
		_, env := call(s.e, "GET", "/api/v1/admin/applications?status=submitted", "", nil)
		if c := codeNum(env); c != int64(code.CodeNoPermission) && c != int64(code.CodeUnauthorized) {
			t.Fatalf("want 5002/2001, got %d", c)
		}
	})

	t.Run("提交申请后查询为待审核", func(t *testing.T) {
		_, env := call(s.e, "POST", "/api/v1/applications", "", map[string]any{
			"inviteCode": inviteA, "nickname": "小满妈妈", "phone": memberPh,
		})
		if codeNum(env) != 0 {
			t.Fatalf("提交申请失败: %v", env)
		}
		app := dataMap(env)
		if app["status"] != "submitted" {
			t.Fatalf("应 submitted: %v", app)
		}
		// 申请者短信登录 → outcome=pending（非成员不发令牌）
		_, env = call(s.e, "POST", "/api/v1/auth/login", "", map[string]any{"phone": memberPh, "code": sms.MockVerifyCode})
		if codeNum(env) != 0 {
			t.Fatalf("登录失败: %v", env)
		}
		if got := dataMap(env)["outcome"]; got != "application_pending" {
			t.Fatalf("应 pending, got %v", got)
		}
		if dataMap(env)["tokens"] != nil {
			t.Fatalf("非成员不得发令牌")
		}
	})

	t.Run("重复申请同手机号被 4203 拒绝", func(t *testing.T) {
		_, env := call(s.e, "POST", "/api/v1/applications", "", map[string]any{
			"inviteCode": inviteB, "nickname": "再来一次", "phone": memberPh,
		})
		if codeNum(env) != int64(code.CodePhoneDuplicate) {
			t.Fatalf("want 4203, got %v", env)
		}
	})

	t.Run("无效邀请码被 3002 拒绝", func(t *testing.T) {
		_, env := call(s.e, "POST", "/api/v1/applications", "", map[string]any{
			"inviteCode": "WMTTESTZZZZ99", "nickname": "别人", "phone": "13900009999",
		})
		if codeNum(env) != int64(code.CodeInviteInvalid) {
			t.Fatalf("want 3002, got %v", env)
		}
	})

	t.Run("管理员查看列表并通过申请", func(t *testing.T) {
		_, env := call(s.e, "GET", "/api/v1/admin/applications?status=submitted&offset=0&limit=10", adminToken, nil)
		if codeNum(env) != 0 {
			t.Fatalf("列表失败: %v", env)
		}
		items := dataMap(env)["items"].([]any)
		if len(items) == 0 {
			t.Fatalf("应有待审核申请")
		}
		first := items[0].(map[string]any)
		id := int64(first["id"].(float64))
		_, env = call(s.e, "POST", fmt.Sprintf("/api/v1/admin/applications/%d/approve", id), adminToken, map[string]any{})
		if codeNum(env) != 0 {
			t.Fatalf("通过失败: %v", env)
		}
	})

	t.Run("审核通过后成员可登录并读取与更新资料", func(t *testing.T) {
		token := s.login(t, memberPh)
		_, env := call(s.e, "GET", "/api/v1/me", token, nil)
		if codeNum(env) != 0 {
			t.Fatalf("获取资料失败: %v", env)
		}
		me := dataMap(env)
		if me["nickname"] != "小满妈妈" || me["maskedPhone"] != "138****1234" {
			t.Fatalf("资料不符: %v", me)
		}
		_, env = call(s.e, "PATCH", "/api/v1/me", token, map[string]any{"nickname": "满妈"})
		if codeNum(env) != 0 {
			t.Fatalf("更新资料失败: %v", env)
		}
		if dataMap(env)["nickname"] != "满妈" {
			t.Fatalf("昵称未生效: %v", dataMap(env))
		}
		// 红点：仅布尔（红线）
		_, env = call(s.e, "GET", "/api/v1/me/badges", token, nil)
		badges := dataMap(env)
		for _, k := range []string{"notice", "hollow", "dm"} {
			if _, ok := badges[k].(bool); !ok {
				t.Fatalf("badges 必须是布尔字段: %v", badges)
			}
		}
	})

	t.Run("申请者不可访问管理端点", func(t *testing.T) {
		token := s.login(t, memberPh)
		_, env := call(s.e, "GET", "/api/v1/admin/applications?status=", token, nil)
		if codeNum(env) != int64(code.CodeNoPermission) {
			t.Fatalf("家长角色访问 admin 应 5002, got %v", env)
		}
	})

	t.Run("昵称超 8 字被 1001 拒绝", func(t *testing.T) {
		token := s.login(t, memberPh)
		_, env := call(s.e, "PATCH", "/api/v1/me", token, map[string]any{"nickname": "这个昵称实在是太长了"})
		if codeNum(env) != int64(code.CodeValidation) {
			t.Fatalf("want 1001, got %v", env)
		}
	})

	t.Run("手机号明文仅管理端点可解且写审计", func(t *testing.T) {
		var memberID int64
		hash := s.phone.Hash(memberPh)
		if err := s.db.Raw("SELECT member_id FROM member_secure WHERE phone_hash = ?", hash).
			Scan(&memberID).Error; err != nil || memberID == 0 {
			t.Fatalf("找不到成员: %v %d", err, memberID)
		}
		adminToken := s.login(t, adminPhon)
		_, env := call(s.e, "GET", fmt.Sprintf("/api/v1/admin/members/%d/phone", memberID), adminToken, nil)
		if codeNum(env) != 0 {
			t.Fatalf("明文查看失败: %v", env)
		}
		if dataMap(env)["phone"] != memberPh {
			t.Fatalf("明文不符: %v", dataMap(env))
		}
		var n int64
		if err := s.db.Raw("SELECT COUNT(*) FROM audit_log WHERE action = 'phone.view' AND target_id = ?", memberID).
			Scan(&n).Error; err != nil || n == 0 {
			t.Fatalf("phone.view 审计缺失: %v %d", err, n)
		}
	})
}
