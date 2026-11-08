//go:build integration

// 树洞集成测试：匿名 DTO、回应 toggle、举报匿名、删帖级联、危机卡（开发计划 M2）。
package it

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

)

func submitAndApprove(t *testing.T, s *itEnv, adminToken, invite, nickname, phone string) {
	t.Helper()
	_, env := call(s.e, "POST", "/api/v1/applications", "", map[string]any{
		"inviteCode": invite, "nickname": nickname, "phone": phone,
	})
	if codeNum(env) != 0 {
		t.Fatalf("提交申请失败: %v", env)
	}
	appID := int64(dataMap(env)["id"].(float64))
	_, env = call(s.e, "POST", fmt.Sprintf("/api/v1/admin/applications/%d/approve", appID), adminToken, map[string]any{})
	if codeNum(env) != 0 {
		t.Fatalf("审核失败: %v", env)
	}
}

func TestHollowFlow(t *testing.T) {
	s := setup(t)
	adminToken := s.seedAdmin(t)
	s.seedInvite(t, "WMTTESTAAAA1", 2) // 两个成员各用一次

	submitAndApprove(t, s, adminToken, "WMTTESTAAAA1", "小满妈妈", memberPh)
	submitAndApprove(t, s, adminToken, "WMTTESTAAAA1", "阿柚", member2Ph)
	memberToken := s.login(t, memberPh)
	otherToken := s.login(t, member2Ph)

	var postID int64

	t.Run("匿名发帖：DTO 不含作者字段", func(t *testing.T) {
		_, env := call(s.e, "POST", "/api/v1/posts", memberToken, map[string]any{
			"mood": "tired", "title": "还是没敢跟老师说", "body": "每天绕到河边坐两小时", "isAnonymous": true,
		})
		if codeNum(env) != 0 {
			t.Fatalf("发帖失败: %v", env)
		}
		data := dataMap(env)
		postID = int64(data["id"].(float64))
		if data["displayName"] != "一位家长" {
			t.Fatalf("匿名显示名不符: %v", data["displayName"])
		}
		payload, _ := json.Marshal(env)
		if strings.Contains(string(payload), "author") || strings.Contains(string(payload), "小满妈妈") {
			t.Fatalf("匿名响应泄露作者信息: %s", payload)
		}
	})

	t.Run("他人回应：楼层 + 作者红点", func(t *testing.T) {
		_, env := call(s.e, "POST", fmt.Sprintf("/api/v1/posts/%d/floors", postID), otherToken, map[string]any{
			"content": "我先接住你", "isAnonymous": false,
		})
		if codeNum(env) != 0 {
			t.Fatalf("回应失败: %v", env)
		}
		// 帖子作者红点（仅布尔）
		_, env = call(s.e, "GET", "/api/v1/me/badges", memberToken, nil)
		badges := dataMap(env)
		if v, ok := badges["hollow"].(bool); !ok || !v {
			t.Fatalf("作者应有 hollow 红点: %v", badges)
		}
	})

	t.Run("回应 toggle：抱一下再取消", func(t *testing.T) {
		_, env := call(s.e, "POST", fmt.Sprintf("/api/v1/posts/%d/reactions", postID), otherToken, map[string]any{"type": "hug"})
		if codeNum(env) != 0 {
			t.Fatalf("hug 失败: %v", env)
		}
		st := dataMap(env)
		if st["active"] != true || st["count"].(float64) != 1 {
			t.Fatalf("首次 hug 应 active+1: %v", st)
		}
		_, env = call(s.e, "POST", fmt.Sprintf("/api/v1/posts/%d/reactions", postID), otherToken, map[string]any{"type": "hug"})
		st = dataMap(env)
		if st["active"] != false || st["count"].(float64) != 0 {
			t.Fatalf("再次 hug 应取消: %v", st)
		}
	})

	t.Run("危机词不拦截且详情带求助标记", func(t *testing.T) {
		_, env := call(s.e, "POST", "/api/v1/posts", memberToken, map[string]any{
			"mood": "tired", "title": "最近很累", "body": "有时候真的不想活了", "isAnonymous": true,
		})
		if codeNum(env) != 0 {
			t.Fatalf("危机内容必须不拦截（红线）: %v", env)
		}
		crisisID := int64(dataMap(env)["id"].(float64))
		_, env = call(s.e, "GET", fmt.Sprintf("/api/v1/posts/%d", crisisID), memberToken, nil)
		if codeNum(env) != 0 {
			t.Fatalf("详情失败: %v", env)
		}
		if dataMap(env)["crisis"] != true {
			t.Fatalf("详情应带 crisis=true（求助卡）: %v", dataMap(env)["crisis"])
		}
	})

	t.Run("举报：reporter 身份不回传", func(t *testing.T) {
		_, env := call(s.e, "POST", "/api/v1/reports", otherToken, map[string]any{
			"targetType": "post", "targetId": postID, "reason": "attack",
		})
		if codeNum(env) != 0 {
			t.Fatalf("举报失败: %v", env)
		}
		_, env = call(s.e, "GET", "/api/v1/admin/reports?status=open", adminToken, nil)
		if codeNum(env) != 0 {
			t.Fatalf("队列失败: %v", env)
		}
		payload, _ := json.Marshal(env)
		if strings.Contains(string(payload), "reporter") {
			t.Fatalf("举报响应不得回传举报人信息: %s", payload)
		}
		items := dataMap(env)["items"].([]any)
		if len(items) == 0 {
			t.Fatalf("队列应有待处理举报")
		}
	})

	t.Run("管理员处置 remove_content：帖级联隐藏", func(t *testing.T) {
		_, env := call(s.e, "GET", "/api/v1/admin/reports?status=open", adminToken, nil)
		items := dataMap(env)["items"].([]any)
		repID := int64(items[0].(map[string]any)["id"].(float64))
		_, env = call(s.e, "POST", fmt.Sprintf("/api/v1/admin/reports/%d/handle", repID), adminToken, map[string]any{
			"action": "remove_content",
		})
		if codeNum(env) != 0 {
			t.Fatalf("处置失败: %v", env)
		}
		_, env = call(s.e, "GET", fmt.Sprintf("/api/v1/posts/%d", postID), memberToken, nil)
		if codeNum(env) != 5001 {
			t.Fatalf("下架后详情应 5001, got %v", env)
		}
	})

	t.Run("我的树洞：匿名帖显示为自己的记录", func(t *testing.T) {
		_, env := call(s.e, "GET", "/api/v1/me/posts", memberToken, nil)
		if codeNum(env) != 0 {
			t.Fatalf("我的树洞失败: %v", env)
		}
		items := dataMap(env)["items"].([]any)
		if len(items) == 0 {
			t.Fatalf("应含自己的匿名帖")
		}
		first := items[0].(map[string]any)
		if first["displayName"] != "小满妈妈" {
			t.Fatalf("我的树洞应显示本人昵称: %v", first["displayName"])
		}
		if first["isAnonymous"] != true {
			t.Fatalf("对外匿名标记应保留: %v", first)
		}
	})
}
