package utils

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// GenerateTokenPair 生成一对 Token
func GenerateTokenPair(userID int, secret string) (string, string, error) {
	// Access Token (1 小时)
	atClaims := jwt.MapClaims{
		"userID": userID,
		"type":   TokenTypeAccess,
		"exp":    time.Now().Add(time.Hour * 1).Unix(),
	}
	at, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims).SignedString([]byte(secret))

	// Refresh Token (7 天)
	rtClaims := jwt.MapClaims{
		"userID": userID,
		"type":   TokenTypeRefresh,
		"exp":    time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	rt, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims).SignedString([]byte(secret))

	return at, rt, nil
}
