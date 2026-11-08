// Package cursor 时间线游标分页（架构设计 4.4：时间线一律时间倒序，游标 = 时间+ID）。
package cursor

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Key 序列化游标（base64("unixNano|id")）。
func Key(at time.Time, id int64) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d|%d", at.UnixNano(), id)))
}

// Parse 还原游标；空串返回零值与 false（首页）。
func Parse(s string) (time.Time, int64, bool, error) {
	if s == "" {
		return time.Time{}, 0, false, nil
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, 0, false, err
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, false, fmt.Errorf("cursor: 非法格式")
	}
	nano, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, 0, false, err
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, false, err
	}
	return time.Unix(0, nano), id, true, nil
}
