package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Environment    string `mapstructure:"environment"`
	Port           string `mapstructure:"port"`
	LogLevel       string `mapstructure:"log_level"`
	
	// Redis配置
	RedisURL       string `mapstructure:"redis_url"`
	
	// JWT配置
	JWTSecret      string `mapstructure:"jwt_secret"`
	
	// 服务发现配置
	Services       map[string]ServiceConfig `mapstructure:"services"`
	
	// 限流配置
	RateLimit      RateLimitConfig `mapstructure:"rate_limit"`
	
	// 超时配置
	Timeout        TimeoutConfig `mapstructure:"timeout"`
	
	// CORS配置
	CORS           CORSConfig `mapstructure:"cors"`
	
	// 监控配置
	Metrics        MetricsConfig `mapstructure:"metrics"`
	
	// 健康检查配置
	HealthCheck    HealthCheckConfig `mapstructure:"health_check"`
}

type ServiceConfig struct {
	URL             string `mapstructure:"url"`
	Timeout         int    `mapstructure:"timeout"`         // 超时时间（秒）
	MaxRetries      int    `mapstructure:"max_retries"`     // 最大重试次数
	HealthEndpoint  string `mapstructure:"health_endpoint"` // 健康检查端点
}

type RateLimitConfig struct {
	Enabled       bool `mapstructure:"enabled"`
	DefaultRPS    int  `mapstructure:"default_rps"`    // 每秒请求数
	DefaultBurst  int  `mapstructure:"default_burst"`  // 突发请求数
	UserRPS       int  `mapstructure:"user_rps"`       // 每用户每秒请求数
	IPRateLimit   int  `mapstructure:"ip_rate_limit"`  // 每IP限制
}

type TimeoutConfig struct {
	Read    int `mapstructure:"read"`    // 读取超时（秒）
	Write   int `mapstructure:"write"`   // 写入超时（秒）
	Idle    int `mapstructure:"idle"`    // 空闲超时（秒）
	Handler int `mapstructure:"handler"` // 处理超时（秒）
}

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	ExposedHeaders   []string `mapstructure:"exposed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
	Port    string `mapstructure:"port"`
}

type HealthCheckConfig struct {
	Enabled  bool `mapstructure:"enabled"`
	Interval int  `mapstructure:"interval"` // 检查间隔（秒）
	Timeout  int  `mapstructure:"timeout"`  // 检查超时（秒）
}

func Load() *Config {
	config := &Config{}

	// 设置配置文件路径和名称
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/api-gateway")

	// 设置环境变量前缀
	viper.SetEnvPrefix("GATEWAY")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，只使用环境变量和默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		}
	}

	// 解析配置
	if err := viper.Unmarshal(config); err != nil {
		panic(err)
	}

	// 从环境变量覆盖敏感配置
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		config.RedisURL = redisURL
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.JWTSecret = jwtSecret
	}

	return config
}

func setDefaults() {
	viper.SetDefault("environment", "development")
	viper.SetDefault("port", "8000")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("redis_url", "redis://localhost:6379")
	viper.SetDefault("jwt_secret", "dev-jwt-secret-change-in-production")
	
	// 服务默认配置
	viper.SetDefault("services.iam-service.url", "http://localhost:8001")
	viper.SetDefault("services.iam-service.timeout", 30)
	viper.SetDefault("services.iam-service.max_retries", 3)
	viper.SetDefault("services.iam-service.health_endpoint", "/health")
	
	viper.SetDefault("services.tenant-service.url", "http://localhost:8002")
	viper.SetDefault("services.tenant-service.timeout", 30)
	viper.SetDefault("services.tenant-service.max_retries", 3)
	viper.SetDefault("services.tenant-service.health_endpoint", "/health")
	
	viper.SetDefault("services.project-service.url", "http://localhost:8003")
	viper.SetDefault("services.project-service.timeout", 30)
	viper.SetDefault("services.project-service.max_retries", 3)
	viper.SetDefault("services.project-service.health_endpoint", "/health")
	
	// 限流默认配置
	viper.SetDefault("rate_limit.enabled", true)
	viper.SetDefault("rate_limit.default_rps", 1000)
	viper.SetDefault("rate_limit.default_burst", 2000)
	viper.SetDefault("rate_limit.user_rps", 100)
	viper.SetDefault("rate_limit.ip_rate_limit", 1000)
	
	// 超时默认配置
	viper.SetDefault("timeout.read", 30)
	viper.SetDefault("timeout.write", 30)
	viper.SetDefault("timeout.idle", 120)
	viper.SetDefault("timeout.handler", 60)
	
	// CORS默认配置
	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"})
	viper.SetDefault("cors.allowed_headers", []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Request-ID"})
	viper.SetDefault("cors.exposed_headers", []string{"Content-Length", "X-Request-ID"})
	viper.SetDefault("cors.allow_credentials", true)
	viper.SetDefault("cors.max_age", 86400)
	
	// 监控默认配置
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.path", "/metrics")
	viper.SetDefault("metrics.port", "9090")
	
	// 健康检查默认配置
	viper.SetDefault("health_check.enabled", true)
	viper.SetDefault("health_check.interval", 30)
	viper.SetDefault("health_check.timeout", 10)
}