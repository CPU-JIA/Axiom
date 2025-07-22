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
	Environment string         `mapstructure:"environment"`
	Port        string         `mapstructure:"port"`
	LogLevel    string         `mapstructure:"log_level"`
	Database    DatabaseConfig `mapstructure:"database"`
	JWT         JWTConfig      `mapstructure:"jwt"`
	CORS        CORSConfig     `mapstructure:"cors"`
	Git         GitConfig      `mapstructure:"git"`
	Webhook     WebhookConfig  `mapstructure:"webhook"`
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

// GitConfig Git配置
type GitConfig struct {
	RepositoryRoot  string `mapstructure:"repository_root"`
	EnableHTTP      bool   `mapstructure:"enable_http"`
	EnableSSH       bool   `mapstructure:"enable_ssh"`
	SSHPort         string `mapstructure:"ssh_port"`
	SSHHostKey      string `mapstructure:"ssh_host_key"`
	MaxFileSize     int64  `mapstructure:"max_file_size"`     // 最大文件大小 (MB)
	MaxRepositorySize int64 `mapstructure:"max_repository_size"` // 最大仓库大小 (MB)
	EnableLFS       bool   `mapstructure:"enable_lfs"`
	LFSStorage      string `mapstructure:"lfs_storage"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	MaxRetries      int    `mapstructure:"max_retries"`
	RetryInterval   int    `mapstructure:"retry_interval"` // 重试间隔 (秒)
	Timeout         int    `mapstructure:"timeout"`        // 超时时间 (秒)
	MaxPayloadSize  int64  `mapstructure:"max_payload_size"` // 最大载荷大小 (KB)
	EnableSignature bool   `mapstructure:"enable_signature"` // 启用签名验证
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
	viper.SetDefault("port", "8004")
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

	// Git设置
	viper.SetDefault("git.repository_root", "/data/repositories")
	viper.SetDefault("git.enable_http", true)
	viper.SetDefault("git.enable_ssh", true)
	viper.SetDefault("git.ssh_port", "2222")
	viper.SetDefault("git.ssh_host_key", "/etc/ssh/ssh_host_rsa_key")
	viper.SetDefault("git.max_file_size", 100)      // 100MB
	viper.SetDefault("git.max_repository_size", 2048) // 2GB
	viper.SetDefault("git.enable_lfs", true)
	viper.SetDefault("git.lfs_storage", "/data/lfs")

	// Webhook设置
	viper.SetDefault("webhook.max_retries", 3)
	viper.SetDefault("webhook.retry_interval", 60)
	viper.SetDefault("webhook.timeout", 30)
	viper.SetDefault("webhook.max_payload_size", 1024) // 1MB
	viper.SetDefault("webhook.enable_signature", true)
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

	if config.Git.RepositoryRoot == "" {
		return fmt.Errorf("Git仓库根目录不能为空")
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

// GetLogLevel 获取日志级别
func (c *Config) GetLogLevel() string {
	return c.LogLevel
}

// LoadFromEnv 从环境变量加载配置（容器化部署使用）
func LoadFromEnv() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "production"),
		Port:        getEnv("PORT", "8004"),
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
		CORS: CORSConfig{
			AllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ","),
			AllowedMethods: strings.Split(getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"), ","),
			AllowedHeaders: strings.Split(getEnv("CORS_ALLOWED_HEADERS", "*"), ","),
		},
		Git: GitConfig{
			RepositoryRoot:    getEnv("GIT_REPOSITORY_ROOT", "/data/repositories"),
			EnableHTTP:        getEnvAsBool("GIT_ENABLE_HTTP", true),
			EnableSSH:         getEnvAsBool("GIT_ENABLE_SSH", true),
			SSHPort:           getEnv("GIT_SSH_PORT", "2222"),
			SSHHostKey:        getEnv("GIT_SSH_HOST_KEY", "/etc/ssh/ssh_host_rsa_key"),
			MaxFileSize:       getEnvAsInt64("GIT_MAX_FILE_SIZE", 100),
			MaxRepositorySize: getEnvAsInt64("GIT_MAX_REPOSITORY_SIZE", 2048),
			EnableLFS:         getEnvAsBool("GIT_ENABLE_LFS", true),
			LFSStorage:        getEnv("GIT_LFS_STORAGE", "/data/lfs"),
		},
		Webhook: WebhookConfig{
			MaxRetries:      getEnvAsInt("WEBHOOK_MAX_RETRIES", 3),
			RetryInterval:   getEnvAsInt("WEBHOOK_RETRY_INTERVAL", 60),
			Timeout:         getEnvAsInt("WEBHOOK_TIMEOUT", 30),
			MaxPayloadSize:  getEnvAsInt64("WEBHOOK_MAX_PAYLOAD_SIZE", 1024),
			EnableSignature: getEnvAsBool("WEBHOOK_ENABLE_SIGNATURE", true),
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

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	var value int64
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