package config

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	MySQL     MySQLConfig     `mapstructure:"mysql"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Etcd      EtcdConfig      `mapstructure:"etcd"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type ServerConfig struct {
	HttpPort int `mapstructure:"http_port"`
	GrpcPort int `mapstructure:"grpc_port"`
	// 对应配置文件中的 internal_ip，同时可以被环境变量覆盖
	InternalIP string `mapstructure:"internal_ip"`
}

type MySQLConfig struct {
	DSN          string `mapstructure:"dsn"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`      // 最大连接数
	MinIdleConns int    `mapstructure:"min_idle_conns"` // 最小空闲连接数
}

type EtcdConfig struct {
	Endpoints []string `mapstructure:"endpoints"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"` // 过期时间（小时）
}

type RateLimitConfig struct {
	Enable bool `mapstructure:"enable"`
	// 策略映射表：Key 为接口路径，Value 为每分钟允许次数
	// 例如："/graphql": 100, "/api/v1/login": 5
	Strategies map[string]int `mapstructure:"strategies"`
	Default    int            `mapstructure:"default"` // 没配置时的默认频率
}
