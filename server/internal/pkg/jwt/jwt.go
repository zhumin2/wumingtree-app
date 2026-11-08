// Package jwt 双令牌签发与解析。
//
// access 7 天 + refresh 30 天；refresh 一次性轮换，复用检测在 service 层
// （旧 refresh 再次使用即吊销该成员全部会话 → code.CodeTokenReuse）。
package jwt

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	AccessTTL  = 7 * 24 * time.Hour
	RefreshTTL = 30 * 24 * time.Hour

	TypAccess  = "access"
	TypRefresh = "refresh"
)

var (
	ErrTokenInvalid = errors.New("jwt: token invalid")
	ErrTokenExpired = errors.New("jwt: token expired")
)

// Claims 自定义载荷。sub 承载成员 ID（ RegisteredClaims.Subject 为字符串）。
type Claims struct {
	Role string `json:"role"`
	Typ  string `json:"typ"` // access / refresh
	JTI  string `json:"jti"` // refresh 轮换与吊销的凭据
	jwt.RegisteredClaims
}

// Issue 签发令牌，返回 token 串与 jti（service 层需保存 refresh 的 jti 用于轮换检测）。
func Issue(secret string, memberID int64, role, typ string, ttl time.Duration) (string, string, error) {
	jti := uuid.NewString()
	now := time.Now()
	claims := &Claims{
		Role: role,
		Typ:  typ,
		JTI:  jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   formatID(memberID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "wumingtree",
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return s, jti, err
}

// Parse 校验并解析令牌。过期返回 ErrTokenExpired，其余无效情形返回 ErrTokenInvalid。
func Parse(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if claims.Typ != TypAccess && claims.Typ != TypRefresh {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}
