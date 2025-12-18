package config

import (
	"fmt"
	"github.com/spf13/viper"
)

func LoadConfig(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.AutomaticEnv() // 允许通过环境变量覆盖配置，如 MYSQL_DSN

	// 1. 显式绑定：将环境变量 MY_POD_IP 映射到结构体中的 server.internal_ip
	// 这样无论配置文件里写什么，只要有 MY_POD_IP 环境变量，它就会被覆盖
	_ = v.BindEnv("server.internal_ip", "MY_POD_IP")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
