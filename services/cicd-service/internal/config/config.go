package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Environment string            `mapstructure:"environment"`
	Port        string            `mapstructure:"port"`
	LogLevel    string            `mapstructure:"log_level"`
	Database    DatabaseConfig    `mapstructure:"database"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	CORS        CORSConfig        `mapstructure:"cors"`
	Kubernetes  KubernetesConfig  `mapstructure:"kubernetes"`
	Tekton      TektonConfig      `mapstructure:"tekton"`
	Storage     StorageConfig     `mapstructure:"storage"`
	Cache       CacheConfig       `mapstructure:"cache"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	Notification NotificationConfig `mapstructure:"notification"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret            string `mapstructure:"secret"`
	AccessTokenExpiry int    `mapstructure:"access_token_expiry"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

// KubernetesConfig Kubernetes配置
type KubernetesConfig struct {
	ConfigPath      string `mapstructure:"config_path"`
	InCluster       bool   `mapstructure:"in_cluster"`
	Namespace       string `mapstructure:"namespace"`
	ServiceAccount  string `mapstructure:"service_account"`
	ImagePullSecret string `mapstructure:"image_pull_secret"`
}

// TektonConfig Tekton配置
type TektonConfig struct {
	Namespace         string `mapstructure:"namespace"`
	DefaultTimeout    int    `mapstructure:"default_timeout"`    // 默认超时时间(秒)
	PipelineRunTTL    int    `mapstructure:"pipeline_run_ttl"`   // PipelineRun保留时间(小时)
	TaskRunTTL        int    `mapstructure:"task_run_ttl"`       // TaskRun保留时间(小时)
	MaxConcurrentRuns int    `mapstructure:"max_concurrent_runs"` // 最大并发运行数
	ResourceQuota     ResourceQuotaConfig `mapstructure:"resource_quota"`
}

// ResourceQuotaConfig 资源配额配置
type ResourceQuotaConfig struct {
	DefaultCPU    string `mapstructure:"default_cpu"`
	DefaultMemory string `mapstructure:"default_memory"`
	MaxCPU        string `mapstructure:"max_cpu"`
	MaxMemory     string `mapstructure:"max_memory"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type          string `mapstructure:"type"`           // local, s3, nfs
	LocalPath     string `mapstructure:"local_path"`     // 本地存储路径
	S3Config      S3Config `mapstructure:"s3"`           // S3配置
	RetentionDays int    `mapstructure:"retention_days"` // 日志保留天数
}

// S3Config S3存储配置
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	UseSSL          bool   `mapstructure:"use_ssl"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type           string `mapstructure:"type"`            // local, redis, memory
	RedisURL       string `mapstructure:"redis_url"`       // Redis连接URL
	LocalPath      string `mapstructure:"local_path"`      // 本地缓存路径
	MaxSizeGB      int    `mapstructure:"max_size_gb"`     // 最大缓存大小(GB)
	TTLHours       int    `mapstructure:"ttl_hours"`       // 缓存过期时间(小时)
	CleanupInterval int   `mapstructure:"cleanup_interval"` // 清理间隔(分钟)
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level         string `mapstructure:"level"`
	Format        string `mapstructure:"format"`         // json, text
	Output        string `mapstructure:"output"`         // stdout, file
	FilePath      string `mapstructure:"file_path"`      // 日志文件路径
	MaxSize       int    `mapstructure:"max_size"`       // 最大文件大小(MB)
	MaxBackups    int    `mapstructure:"max_backups"`    // 最大备份数
	MaxAge        int    `mapstructure:"max_age"`        // 最大保留天数
	Compress      bool   `mapstructure:"compress"`       // 是否压缩
	EnableConsole bool   `mapstructure:"enable_console"` // 启用控制台输出
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	WebhookURL     string   `mapstructure:"webhook_url"`     // Webhook通知URL
	SlackToken     string   `mapstructure:"slack_token"`     // Slack Bot Token
	EmailSMTP      SMTPConfig `mapstructure:"email_smtp"`    // 邮件SMTP配置
	EnabledChannels []string `mapstructure:"enabled_channels"` // 启用的通知渠道
}

// SMTPConfig SMTP邮件配置
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	UseTLS   bool   `mapstructure:"use_tls"`
}

// Load 加载配置
func Load() *Config {
	config := &Config{}

	// 设置配置文件名和路径
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// 环境变量替换
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("配置文件未找到，使用环境变量和默认值")
		} else {
			log.Fatalf("读取配置文件失败: %v", err)
		}
	}

	// 解析配置到结构体
	if err := viper.Unmarshal(config); err != nil {
		log.Fatalf("配置解析失败: %v", err)
	}

	// 验证必要配置
	if err := validateConfig(config); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	return config
}

// setDefaults 设置默认配置值
func setDefaults() {
	// 应用设置
	viper.SetDefault("environment", "development")
	viper.SetDefault("port", "8005")
	viper.SetDefault("log_level", "info")

	// 数据库设置
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "password")
	viper.SetDefault("database.dbname", "euclid_elements")
	viper.SetDefault("database.sslmode", "disable")

	// JWT设置
	viper.SetDefault("jwt.secret", "your-secret-key")
	viper.SetDefault("jwt.access_token_expiry", 3600)

	// CORS设置
	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allowed_headers", []string{"*"})

	// Kubernetes设置
	viper.SetDefault("kubernetes.config_path", "")
	viper.SetDefault("kubernetes.in_cluster", false)
	viper.SetDefault("kubernetes.namespace", "cicd")
	viper.SetDefault("kubernetes.service_account", "cicd-service")
	viper.SetDefault("kubernetes.image_pull_secret", "")

	// Tekton设置
	viper.SetDefault("tekton.namespace", "tekton-pipelines")
	viper.SetDefault("tekton.default_timeout", 3600)
	viper.SetDefault("tekton.pipeline_run_ttl", 168) // 7天
	viper.SetDefault("tekton.task_run_ttl", 24)      // 1天
	viper.SetDefault("tekton.max_concurrent_runs", 10)
	viper.SetDefault("tekton.resource_quota.default_cpu", "100m")
	viper.SetDefault("tekton.resource_quota.default_memory", "128Mi")
	viper.SetDefault("tekton.resource_quota.max_cpu", "2")
	viper.SetDefault("tekton.resource_quota.max_memory", "4Gi")

	// 存储设置
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.local_path", "/data/cicd")
	viper.SetDefault("storage.retention_days", 30)

	// 缓存设置
	viper.SetDefault("cache.type", "local")
	viper.SetDefault("cache.local_path", "/data/cache")
	viper.SetDefault("cache.max_size_gb", 10)
	viper.SetDefault("cache.ttl_hours", 168) // 7天
	viper.SetDefault("cache.cleanup_interval", 60) // 1小时

	// 日志设置
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
	viper.SetDefault("logging.compress", true)
	viper.SetDefault("logging.enable_console", true)

	// 通知设置
	viper.SetDefault("notification.enabled_channels", []string{})
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	if config.Database.Host == "" {
		return fmt.Errorf("数据库主机不能为空")
	}

	if config.Database.DBName == "" {
		return fmt.Errorf("数据库名称不能为空")
	}

	if config.JWT.Secret == "" || config.JWT.Secret == "your-secret-key" {
		return fmt.Errorf("JWT密钥不能为空或使用默认值")
	}

	if config.Kubernetes.Namespace == "" {
		return fmt.Errorf("Kubernetes命名空间不能为空")
	}

	if config.Storage.Type == "local" && config.Storage.LocalPath == "" {
		return fmt.Errorf("本地存储路径不能为空")
	}

	return nil
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// IsDevelopment 是否为开发环境
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction 是否为生产环境
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// LoadFromEnv 从环境变量加载配置（容器化部署使用）
func LoadFromEnv() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "production"),
		Port:        getEnv("PORT", "8005"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "euclid_elements"),
			SSLMode:  getEnv("DB_SSL_MODE", "require"),
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", ""),
			AccessTokenExpiry: getEnvAsInt("JWT_ACCESS_TOKEN_EXPIRY", 3600),
		},
		Kubernetes: KubernetesConfig{
			ConfigPath:      getEnv("K8S_CONFIG_PATH", ""),
			InCluster:       getEnvAsBool("K8S_IN_CLUSTER", true),
			Namespace:       getEnv("K8S_NAMESPACE", "cicd"),
			ServiceAccount:  getEnv("K8S_SERVICE_ACCOUNT", "cicd-service"),
			ImagePullSecret: getEnv("K8S_IMAGE_PULL_SECRET", ""),
		},
		Tekton: TektonConfig{
			Namespace:         getEnv("TEKTON_NAMESPACE", "tekton-pipelines"),
			DefaultTimeout:    getEnvAsInt("TEKTON_DEFAULT_TIMEOUT", 3600),
			PipelineRunTTL:    getEnvAsInt("TEKTON_PIPELINE_RUN_TTL", 168),
			TaskRunTTL:        getEnvAsInt("TEKTON_TASK_RUN_TTL", 24),
			MaxConcurrentRuns: getEnvAsInt("TEKTON_MAX_CONCURRENT_RUNS", 10),
			ResourceQuota: ResourceQuotaConfig{
				DefaultCPU:    getEnv("TEKTON_DEFAULT_CPU", "100m"),
				DefaultMemory: getEnv("TEKTON_DEFAULT_MEMORY", "128Mi"),
				MaxCPU:        getEnv("TEKTON_MAX_CPU", "2"),
				MaxMemory:     getEnv("TEKTON_MAX_MEMORY", "4Gi"),
			},
		},
		Storage: StorageConfig{
			Type:          getEnv("STORAGE_TYPE", "local"),
			LocalPath:     getEnv("STORAGE_LOCAL_PATH", "/data/cicd"),
			RetentionDays: getEnvAsInt("STORAGE_RETENTION_DAYS", 30),
		},
		Cache: CacheConfig{
			Type:            getEnv("CACHE_TYPE", "local"),
			LocalPath:       getEnv("CACHE_LOCAL_PATH", "/data/cache"),
			MaxSizeGB:       getEnvAsInt("CACHE_MAX_SIZE_GB", 10),
			TTLHours:        getEnvAsInt("CACHE_TTL_HOURS", 168),
			CleanupInterval: getEnvAsInt("CACHE_CLEANUP_INTERVAL", 60),
		},
	}
}

// 辅助函数
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	return valueStr == "true" || valueStr == "1" || valueStr == "yes"
}