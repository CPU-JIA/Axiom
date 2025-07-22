package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Environment    string `mapstructure:"environment"`
	Port           string `mapstructure:"port"`
	DatabaseURL    string `mapstructure:"database_url"`
	RedisURL       string `mapstructure:"redis_url"`
	JWTSecret      string `mapstructure:"jwt_secret"`
	InternalSecret string `mapstructure:"internal_secret"`
	LogLevel       string `mapstructure:"log_level"`
	
	// SMTP邮件配置
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	
	// MFA配置
	MFAIssuer string `mapstructure:"mfa_issuer"`
	
	// 安全配置
	PasswordMinLength int `mapstructure:"password_min_length"`
	MaxLoginAttempts  int `mapstructure:"max_login_attempts"`
	LockoutDuration   int `mapstructure:"lockout_duration"` // 分钟
	
	// Token配置
	AccessTokenExpiry  int `mapstructure:"access_token_expiry"`  // 分钟
	RefreshTokenExpiry int `mapstructure:"refresh_token_expiry"` // 天
	
	// 文件上传配置
	MaxUploadSize int64  `mapstructure:"max_upload_size"` // bytes
	UploadPath    string `mapstructure:"upload_path"`
}

func Load() *Config {
	config := &Config{}

	// 设置配置文件路径和名称
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/iam-service")

	// 设置环境变量前缀
	viper.SetEnvPrefix("IAM")
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
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.DatabaseURL = dbURL
	}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		config.RedisURL = redisURL
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.JWTSecret = jwtSecret
	}
	if internalSecret := os.Getenv("INTERNAL_SECRET"); internalSecret != "" {
		config.InternalSecret = internalSecret
	}

	return config
}

func setDefaults() {
	viper.SetDefault("environment", "development")
	viper.SetDefault("port", "8001")
	viper.SetDefault("database_url", "postgres://developer:dev_password_2024@localhost:5432/cloud_platform?sslmode=disable")
	viper.SetDefault("redis_url", "redis://localhost:6379")
	viper.SetDefault("jwt_secret", "dev-jwt-secret-change-in-production")
	viper.SetDefault("internal_secret", "dev-internal-secret-change-in-production")
	viper.SetDefault("log_level", "info")
	
	// SMTP默认配置
	viper.SetDefault("smtp_host", "localhost")
	viper.SetDefault("smtp_port", 587)
	viper.SetDefault("smtp_user", "")
	viper.SetDefault("smtp_password", "")
	
	// MFA默认配置
	viper.SetDefault("mfa_issuer", "Cloud Platform")
	
	// 安全默认配置
	viper.SetDefault("password_min_length", 8)
	viper.SetDefault("max_login_attempts", 5)
	viper.SetDefault("lockout_duration", 15) // 15分钟
	
	// Token默认配置
	viper.SetDefault("access_token_expiry", 60)   // 1小时
	viper.SetDefault("refresh_token_expiry", 30)  // 30天
	
	// 文件上传默认配置
	viper.SetDefault("max_upload_size", 5*1024*1024) // 5MB
	viper.SetDefault("upload_path", "./uploads")
}