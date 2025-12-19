package middleware

import (
	"net/http"
	"strconv"
)

// CacheConfig 缓存配置
type CacheConfig struct {
	MaxAge               int  // 缓存秒数
	Private              bool // 是否为私有缓存 (针对个人资料必选 true)
	StaleWhileRevalidate int  // 过期后允许异步更新的时间
}

// CacheMiddleware 缓存中间件
func CacheMiddleware(cfg CacheConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 只有 GET 请求才走 CDN 缓存
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			cacheType := "public"
			if cfg.Private {
				cacheType = "private"
			}

			// 构造 Cache-Control 指令
			headerVal := cacheType + ", max-age=" + strconv.Itoa(cfg.MaxAge)
			if cfg.StaleWhileRevalidate > 0 {
				headerVal += ", stale-while-revalidate=" + strconv.Itoa(cfg.StaleWhileRevalidate)
			}

			w.Header().Set("Cache-Control", headerVal)

			// 关键：防止 CDN 把 A 用户的资料给 B 用户
			// 告知缓存服务器根据 Authorization 和 Cookie 区分缓存副本
			w.Header().Set("Vary", "Authorization, Cookie")

			next.ServeHTTP(w, r)
		})
	}
}
