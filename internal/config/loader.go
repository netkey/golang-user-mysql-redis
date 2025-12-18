package config

import (
	"fmt"
	"github.com/spf13/viper"
)

func LoadConfig(path string) (*config.Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.AutomaticEnv() // 允许通过环境变量覆盖配置，如 MYSQL_DSN

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
