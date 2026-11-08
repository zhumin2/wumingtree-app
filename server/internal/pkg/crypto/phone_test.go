package crypto

import (
	"bytes"
	"strings"
	"testing"
)

func testKey(t *testing.T) *PhoneCrypto {
	t.Helper()
	p, err := NewPhoneCrypto(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatalf("构造失败: %v", err)
	}
	return p
}

func TestNewPhoneCrypto_KeyLen(t *testing.T) {
	if _, err := NewPhoneCrypto(make([]byte, 16)); err != ErrKeyLen {
		t.Fatalf("want ErrKeyLen, got %v", err)
	}
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	p := testKey(t)
	enc, err := p.Encrypt("13800001234")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if strings.Contains(enc, "13800001234") {
		t.Fatalf("密文不得包含明文")
	}
	// 两次加密密文不同（随机 nonce）
	enc2, _ := p.Encrypt("13800001234")
	if enc == enc2 {
		t.Fatalf("随机 nonce 缺失，密文不应相同")
	}
	got, err := p.Decrypt(enc)
	if err != nil || got != "13800001234" {
		t.Fatalf("解密失败: %v, %s", err, got)
	}
}

func TestDecrypt_Tampered(t *testing.T) {
	p := testKey(t)
	enc, _ := p.Encrypt("13800001234")
	raw := []byte(enc)
	raw[10] ^= 0xFF
	if _, err := p.Decrypt(string(raw)); err != ErrCipherText {
		t.Fatalf("篡改应返回 ErrCipherText, got %v", err)
	}
	// 错误密钥
	other, _ := NewPhoneCrypto(bytes.Repeat([]byte{9}, 32))
	if _, err := other.Decrypt(enc); err != ErrCipherText {
		t.Fatalf("错钥应返回 ErrCipherText, got %v", err)
	}
}

func TestHash_StableAndDistinct(t *testing.T) {
	p := testKey(t)
	h1 := p.Hash("13800001234")
	h2 := p.Hash("13800001234")
	h3 := p.Hash("13800005678")
	if h1 != h2 || len(h1) != 64 {
		t.Fatalf("摘要应稳定且为 64 位 hex")
	}
	if h1 == h3 {
		t.Fatalf("不同号码摘要应不同")
	}
	// 摘要与加密不同钥路：由 Hash 自身决定，不受密文影响
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"13800001234", "13800001234", false},
		{"138 0000 1234", "13800001234", false},
		{"138-0000-1234", "13800001234", false},
		{"+8613800001234", "13800001234", false},
		{"8613800001234", "13800001234", false},
		{"12345678901", "", true},  // 非 1[3-9] 号段
		{"1380001234", "", true},   // 10 位
		{"138000012345", "", true}, // 12 位
		{"", "", true},
	}
	for _, c := range cases {
		got, err := Normalize(c.in)
		if c.wantErr {
			if err == nil {
				t.Fatalf("%q 应报错", c.in)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Fatalf("%q: want %q, got %q, err=%v", c.in, c.want, got, err)
		}
	}
}

func TestMask(t *testing.T) {
	got, err := Mask("13800001234")
	if err != nil || got != "138****1234" {
		t.Fatalf("want 138****1234, got %s, %v", got, err)
	}
	if _, err := Mask("123"); err == nil {
		t.Fatalf("非手机号应报错")
	}
}
