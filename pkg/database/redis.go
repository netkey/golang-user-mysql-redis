package database

import (
	"context"
	"github.com/netkey/golang-user-mysql-redis/internal/config"
	"github.com/redis/go-redis/v9"
	"time"
)

func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,

		// --- 连接池配置 ---
		PoolSize:     cfg.PoolSize,     // 一般设置为 CPU 核数的 10 倍或根据并发量调整
		MinIdleConns: cfg.MinIdleConns, // 保持一定数量的空闲连接，减少新建连接的开销

		DialTimeout:  5 * time.Second, // 连接超时
		ReadTimeout:  3 * time.Second, // 读超时
		WriteTimeout: 3 * time.Second, // 写超时
		PoolTimeout:  4 * time.Second, // 如果连接池满了，等待可用连接的超时时间
		IdleTimeout:  5 * time.Minute, // 空闲连接闲置多久后关闭
	})

	// 检查连接是否可用
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	return rdb, nil
}
