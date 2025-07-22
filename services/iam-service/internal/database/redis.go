package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient 创建Redis客户端
func NewRedisClient(redisURL string) *redis.Client {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse Redis URL: %v", err))
	}

	// 设置连接池参数
	opts.PoolSize = 10
	opts.MinIdleConns = 5
	opts.PoolTimeout = 30 * time.Second
	opts.ConnMaxIdleTime = 5 * time.Minute

	client := redis.NewClient(opts)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	return client
}

// RedisKeys Redis键名常量
const (
	// 用户会话
	KeyUserSession = "session:user:%s"        // session:user:{user_id}
	KeyUserLockout = "lockout:user:%s"        // lockout:user:{email}
	
	// MFA相关
	KeyMFASetup    = "mfa:setup:%s"          // mfa:setup:{user_id}
	KeyMFABackup   = "mfa:backup:%s:%s"      // mfa:backup:{user_id}:{code}
	
	// 邮箱验证
	KeyEmailVerify = "email:verify:%s:%s"    // email:verify:{user_id}:{code}
	
	// 密码重置
	KeyPasswordReset = "password:reset:%s"   // password:reset:{token}
	
	// 限流
	KeyRateLimit = "rate_limit:%s:%s"        // rate_limit:{type}:{identifier}
	
	// Token黑名单
	KeyTokenBlacklist = "token:blacklist:%s" // token:blacklist:{jti}
	
	// 幂等性
	KeyIdempotency = "idempotent:%s"         // idempotent:{key}
)

// BuildKey 构建Redis键名
func BuildKey(pattern string, args ...interface{}) string {
	return fmt.Sprintf(pattern, args...)
}

// HealthRedis 检查Redis健康状态
func HealthRedis(client *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	return client.Ping(ctx).Err()
}