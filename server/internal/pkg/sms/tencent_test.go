package sms

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhumin2/wumingtree-app/server/internal/pkg/code"
	"github.com/zhumin2/wumingtree-app/server/internal/pkg/tc3"
)

func newTencentTest(t *testing.T, handler http.HandlerFunc) (TencentSender, *[]map[string]any) {
	t.Helper()
	var got []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(body, &m)
		got = append(got, m)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	cl := tc3.New(tc3.Credential{SecretID: "id", SecretKey: "key"}, "sms.tencentcloudapi.com", "sms", "ap-guangzhou")
	cl.HTTP = srv.Client()
	cl.BaseURL = srv.URL
	return TencentSender{TC: cl, SdkAppID: "1400006666", SignName: "无名树", TemplateID: "1001"}, &got
}

func TestTencentSender_OK(t *testing.T) {
	sender, reqs := newTencentTest(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Response": map[string]any{
				"SendStatusSet": []map[string]any{{"Code": "Ok", "Message": "send success"}},
			},
		})
	})
	if err := sender.SendVerifyCode(context.Background(), "13800138000", "654321"); err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if len(*reqs) != 1 {
		t.Fatalf("requests = %d", len(*reqs))
	}
	p := (*reqs)[0]
	if p["SmsSdkAppId"] != "1400006666" || p["SignName"] != "无名树" || p["TemplateId"] != "1001" {
		t.Fatalf("payload params mismatch: %v", p)
	}
	phones, _ := p["PhoneNumberSet"].([]any)
	if len(phones) != 1 || phones[0] != "+8613800138000" {
		t.Fatalf("PhoneNumberSet = %v, want [+8613800138000]", phones)
	}
	params, _ := p["TemplateParamSet"].([]any)
	if len(params) != 1 || params[0] != "654321" {
		t.Fatalf("TemplateParamSet = %v", params)
	}
}

func TestTencentSender_StatusNotOk(t *testing.T) {
	sender, _ := newTencentTest(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Response": map[string]any{
				"SendStatusSet": []map[string]any{{
					"Code":    "FailedOperation.TemplateParamSetNotMatchApprovedTemplate",
					"Message": "params not match",
				}},
			},
		})
	})
	err := sender.SendVerifyCode(context.Background(), "13800138000", "654321")
	if !errors.Is(err, code.ErrDependency) {
		t.Fatalf("want ErrDependency, got %v", err)
	}
}

func TestTencentSender_APIError(t *testing.T) {
	sender, _ := newTencentTest(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Response": map[string]any{
				"Error": map[string]any{"Code": "AuthFailure", "Message": "bad credentials"},
			},
		})
	})
	err := sender.SendVerifyCode(context.Background(), "13800138000", "654321")
	if !errors.Is(err, code.ErrDependency) {
		t.Fatalf("want ErrDependency, got %v", err)
	}
}

func TestTencentSender_EmptyStatusSet(t *testing.T) {
	sender, _ := newTencentTest(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"Response": map[string]any{}})
	})
	err := sender.SendVerifyCode(context.Background(), "13800138000", "654321")
	if !errors.Is(err, code.ErrDependency) {
		t.Fatalf("want ErrDependency, got %v", err)
	}
}

func TestMaskPhone(t *testing.T) {
	if got := maskPhone("13800138000"); got != "****8000" {
		t.Fatalf("maskPhone = %s", got)
	}
	if got := maskPhone("123"); got != "****" {
		t.Fatalf("maskPhone short = %s", got)
	}
}
