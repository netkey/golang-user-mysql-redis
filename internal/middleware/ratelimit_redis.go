package middleware

import (
	"context"
	"github.com/netkey/golang-user-mysql-redis/internal/config"
	"github.com/redis/go-redis/v9"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-redis/redis_rate/v10"
	"go.uber.org/zap"
)

type RedisRateLimiter struct {
	limiter *redis_rate.Limiter
	cfg     config.RateLimitConfig
}

func NewRedisRateLimiter(rdb *redis.Client, cfg config.RateLimitConfig) *RedisRateLimiter {
	return &RedisRateLimiter{
		limiter: redis_rate.NewLimiter(rdb),
		cfg:     cfg,
	}
}

// GetClientIP 提取真实客户端 IP
func GetClientIP(r *http.Request) string {
	// 1. 优先获取 True-Client-IP (CDN/WAF 常用)
	ip := r.Header.Get("True-Client-IP")
	if ip != "" {
		return ip
	}

	// 2. 备选 X-Forwarded-For (格式为: client, proxy1, proxy2)
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		parts := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}

	// 3. 最后退回到 RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (rl *RedisRateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 检查全局开关
		if !rl.cfg.Enable {
			next.ServeHTTP(w, r)
			return
		}

		// 2. 获取当前路径对应的频率限制
		limitNum := rl.cfg.Default
		if customLimit, ok := rl.cfg.Strategies[r.URL.Path]; ok {
			limitNum = customLimit
		}

		// 3. 获取 IP
		clientIP := GetClientIP(r)
		// Key 增加 Path 维度，实现针对不同接口独立限流
		key := "limit:" + r.URL.Path + ":" + clientIP

		res, err := rl.limiter.Allow(r.Context(), key, redis_rate.PerMinute(limitNum))
		if err != nil {
			next.ServeHTTP(w, r) // 降级
			return
		}

		if res.Allowed <= 0 {
			http.Error(w, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
