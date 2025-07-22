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
	// 租户缓存
	KeyTenantInfo    = "tenant:info:%s"       // tenant:info:{tenant_id}
	KeyTenantMembers = "tenant:members:%s"    // tenant:members:{tenant_id}
	KeyTenantStats   = "tenant:stats:%s"      // tenant:stats:{tenant_id}
	
	// 用户租户关系缓存
	KeyUserTenants = "user:tenants:%s"        // user:tenants:{user_id}
	KeyUserRoles   = "user:roles:%s:%s"       // user:roles:{user_id}:{tenant_id}
	
	// 邀请相关
	KeyInviteToken = "invite:token:%s"        // invite:token:{token}
	KeyInviteLimit = "invite:limit:%s:%s"     // invite:limit:{tenant_id}:{email}
	
	// 权限缓存
	KeyMemberPermissions = "member:perms:%s:%s" // member:perms:{user_id}:{tenant_id}
	
	// 限流
	KeyTenantRateLimit = "rate:tenant:%s:%s"  // rate:tenant:{tenant_id}:{action}
	KeyUserRateLimit   = "rate:user:%s:%s"    // rate:user:{user_id}:{action}
	
	// 会话相关
	KeyTenantSession = "session:tenant:%s"    // session:tenant:{session_id}
	
	// 统计缓存
	KeyTenantDailyStats = "stats:daily:%s:%s" // stats:daily:{tenant_id}:{date}
	KeySystemStats      = "stats:system:%s"   // stats:system:{date}
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