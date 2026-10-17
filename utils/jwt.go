package utils

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"strconv"
	"time"
)

const jwtIssuer = "go-douyin"

func GenerateToken(userID uint64) (string, error) {
	secret := []byte(jwtIssuer)
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:  strconv.FormatUint(userID, 10),
		Issuer:   jwtIssuer,
		IssuedAt: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(
			now.Add(2 * time.Hour),
		),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
func ParseToken(tokenString string) (uint64, error) {
	// 2. 创建接收 JWT Payload 的结构体
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtIssuer), nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(jwtIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt())
	if err != nil {
		return 0, err
	}
	if !token.Valid {
		return 0, errors.New("无效的 JWT")
	}
	// 4. 从 Subject 中读取用户 ID
	if claims.Subject == "" {
		return 0, errors.New("JWT 缺少用户 ID")
	}
	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return 0, err
	}

	// 防止转换为 uint 时发生溢出
	if userID == 0 {
		return 0, errors.New("无效的用户 ID")
	}

	return userID, nil
}
