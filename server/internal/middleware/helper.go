package middleware

import "strconv"

// parseSubject 把 JWT sub（字符串成员 ID）转回 int64。
func parseSubject(sub string) (int64, error) {
	return strconv.ParseInt(sub, 10, 64)
}
