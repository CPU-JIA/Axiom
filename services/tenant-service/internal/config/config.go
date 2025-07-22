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
	
	// IAM服务配置
	IAMServiceURL string `mapstructure:"iam_service_url"`
	
	// 租户配置
	DefaultMaxMembers   int `mapstructure:"default_max_members"`
	DefaultMaxProjects  int `mapstructure:"default_max_projects"`
	DefaultStorageQuota int `mapstructure:"default_storage_quota"` // GB
	
	// 通知配置
	KafkaBrokers         []string `mapstructure:"kafka_brokers"`
	NotificationTopic    string   `mapstructure:"notification_topic"`
	TenantEventTopic     string   `mapstructure:"tenant_event_topic"`
	
	// 资源限制
	MaxTenantsPerUser    int `mapstructure:"max_tenants_per_user"`
	TenantNameMinLength  int `mapstructure:"tenant_name_min_length"`
	TenantNameMaxLength  int `mapstructure:"tenant_name_max_length"`
}

func Load() *Config {
	config := &Config{}

	// 设置配置文件路径和名称
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/tenant-service")

	// 设置环境变量前缀
	viper.SetEnvPrefix("TENANT")
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
	viper.SetDefault("port", "8002")
	viper.SetDefault("database_url", "postgres://developer:dev_password_2024@localhost:5432/cloud_platform?sslmode=disable")
	viper.SetDefault("redis_url", "redis://localhost:6379")
	viper.SetDefault("jwt_secret", "dev-jwt-secret-change-in-production")
	viper.SetDefault("internal_secret", "dev-internal-secret-change-in-production")
	viper.SetDefault("log_level", "info")
	
	// IAM服务默认配置
	viper.SetDefault("iam_service_url", "http://localhost:8001")
	
	// 租户默认配置
	viper.SetDefault("default_max_members", 50)
	viper.SetDefault("default_max_projects", 10)
	viper.SetDefault("default_storage_quota", 100) // 100GB
	
	// Kafka默认配置
	viper.SetDefault("kafka_brokers", []string{"localhost:9092"})
	viper.SetDefault("notification_topic", "notifications")
	viper.SetDefault("tenant_event_topic", "tenant-events")
	
	// 资源限制默认配置
	viper.SetDefault("max_tenants_per_user", 5)
	viper.SetDefault("tenant_name_min_length", 3)
	viper.SetDefault("tenant_name_max_length", 50)
}