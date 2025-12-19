package utils

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// GenerateToken 生成 JWT Token
func GenerateToken(userID int, secret string, expireHour int) (string, error) {
	// 1. 创建 Claims（载荷），包含用户 ID 和过期时间
	claims := jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(time.Hour * time.Duration(expireHour)).Unix(),
		"iat":    time.Now().Unix(),
	}

	// 2. 使用指定的签名密钥创建 token 对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 3. 签名并获取完整的编码后的字符串 token
	return token.SignedString([]byte(secret))
}
