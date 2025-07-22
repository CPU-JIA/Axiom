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
	viper.SetDefault("port", "8003")
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
		Port:        getEnv("PORT", "8003"),
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