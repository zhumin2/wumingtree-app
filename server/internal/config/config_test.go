package config

import "testing"

func TestLoad_MockDefaults(t *testing.T) {
	t.Setenv("SMS_TYPE", "mock")
	t.Setenv("AUDIT_MODE", "mock")
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.TencentRegion != "ap-guangzhou" {
		t.Fatalf("TencentRegion default = %s", c.TencentRegion)
	}
}

func TestLoad_RemoteRequiresCredentials(t *testing.T) {
	t.Run("sms remote missing", func(t *testing.T) {
		t.Setenv("SMS_TYPE", "remote")
		t.Setenv("AUDIT_MODE", "mock")
		if _, err := Load(); err == nil {
			t.Fatal("want error when SMS_TYPE=remote without credentials")
		}
	})
	t.Run("audit remote missing", func(t *testing.T) {
		t.Setenv("SMS_TYPE", "mock")
		t.Setenv("AUDIT_MODE", "remote")
		if _, err := Load(); err == nil {
			t.Fatal("want error when AUDIT_MODE=remote without credentials")
		}
	})
	t.Run("remote complete", func(t *testing.T) {
		t.Setenv("SMS_TYPE", "remote")
		t.Setenv("AUDIT_MODE", "remote")
		t.Setenv("TENCENT_SECRET_ID", "test-secret-id")
		t.Setenv("TENCENT_SECRET_KEY", "key")
		t.Setenv("TENCENT_REGION", "ap-shanghai")
		t.Setenv("SMS_SDK_APP_ID", "1400006666")
		t.Setenv("SMS_SIGN_NAME", "无名树")
		t.Setenv("SMS_TEMPLATE_ID", "1001")
		c, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if c.TencentRegion != "ap-shanghai" || c.SmsSdkAppID != "1400006666" {
			t.Fatalf("fields not loaded: %+v", c)
		}
	})
}

func TestLoad_InvalidModes(t *testing.T) {
	t.Setenv("SMS_TYPE", "aliyun")
	if _, err := Load(); err == nil {
		t.Fatal("want error for invalid SMS_TYPE")
	}
}
