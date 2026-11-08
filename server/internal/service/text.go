package service

import "strings"

// NormalizeNickname 称呼校验与归一（对外工具：seedadmin 等复用同一规则）。
func NormalizeNickname(n string) (string, error) {
	return normalizeNickname(n)
}

// trimSpaces 去首尾半角/全角空格。
func trimSpaces(s string) string {
	return strings.Trim(s, " \t\n　")
}
