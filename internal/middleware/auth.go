package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// AuthMiddleware 用于 HTTP/GraphQL
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 格式: Bearer <token>
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				next.ServeHTTP(w, r)
				return
			}

			tokenString := parts[1]
			token, _ := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				// 将 userID 注入 Context
				ctx := context.WithValue(r.Context(), UserIDKey, claims["user_id"])
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
