package service

import "strings"

// trimSpaces 去首尾半角/全角空格。
func trimSpaces(s string) string {
	return strings.Trim(s, " \t\n　")
}
