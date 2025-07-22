package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"api-gateway/pkg/logger"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context, key string) bool
	AllowN(ctx context.Context, key string, n int) bool
	Limit(key string) rate.Limit
	Burst(key string) int
}

// RedisRateLimiter 基于Redis的限流器
type RedisRateLimiter struct {
	client   *redis.Client
	logger   logger.Logger
	defaultLimit rate.Limit
	defaultBurst int
	window       time.Duration
}

// NewRedisRateLimiter 创建Redis限流器
func NewRedisRateLimiter(client *redis.Client, rps int, burst int, logger logger.Logger) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:       client,
		logger:       logger,
		defaultLimit: rate.Limit(rps),
		defaultBurst: burst,
		window:       time.Minute, // 1分钟窗口
	}
}

// Allow 检查是否允许请求
func (r *RedisRateLimiter) Allow(ctx context.Context, key string) bool {
	return r.AllowN(ctx, key, 1)
}

// AllowN 检查是否允许N个请求
func (r *RedisRateLimiter) AllowN(ctx context.Context, key string, n int) bool {
	now := time.Now()
	window := now.Truncate(r.window)
	
	redisKey := fmt.Sprintf("rate_limit:%s:%d", key, window.Unix())
	
	// 使用Redis管道提高性能
	pipe := r.client.Pipeline()
	
	// 增加计数
	incrCmd := pipe.IncrBy(ctx, redisKey, int64(n))
	
	// 设置过期时间
	pipe.Expire(ctx, redisKey, r.window+time.Second)
	
	// 执行管道
	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Error("Rate limit redis error", "key", key, "error", err)
		// Redis错误时允许请求，避免影响正常服务
		return true
	}
	
	currentCount := incrCmd.Val()
	limit := int64(r.defaultLimit) * int64(r.window.Seconds())
	
	allowed := currentCount <= limit
	
	if !allowed {
		r.logger.Warn("Rate limit exceeded", 
			"key", key,
			"current", currentCount,
			"limit", limit,
			"window", window,
		)
	}
	
	return allowed
}

// Limit 返回限制速率
func (r *RedisRateLimiter) Limit(key string) rate.Limit {
	return r.defaultLimit
}

// Burst 返回突发容量
func (r *RedisRateLimiter) Burst(key string) int {
	return r.defaultBurst
}

// MemoryRateLimiter 基于内存的限流器（用于开发环境）
type MemoryRateLimiter struct {
	limiters map[string]*rate.Limiter
	limit    rate.Limit
	burst    int
}

// NewMemoryRateLimiter 创建内存限流器
func NewMemoryRateLimiter(rps int, burst int) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		limit:    rate.Limit(rps),
		burst:    burst,
	}
}

// Allow 检查是否允许请求
func (m *MemoryRateLimiter) Allow(ctx context.Context, key string) bool {
	limiter := m.getLimiter(key)
	return limiter.Allow()
}

// AllowN 检查是否允许N个请求
func (m *MemoryRateLimiter) AllowN(ctx context.Context, key string, n int) bool {
	limiter := m.getLimiter(key)
	return limiter.AllowN(time.Now(), n)
}

// Limit 返回限制速率
func (m *MemoryRateLimiter) Limit(key string) rate.Limit {
	return m.limit
}

// Burst 返回突发容量
func (m *MemoryRateLimiter) Burst(key string) int {
	return m.burst
}

// getLimiter 获取或创建限流器
func (m *MemoryRateLimiter) getLimiter(key string) *rate.Limiter {
	if limiter, exists := m.limiters[key]; exists {
		return limiter
	}
	
	limiter := rate.NewLimiter(m.limit, m.burst)
	m.limiters[key] = limiter
	return limiter
}

// RateLimitKey 生成限流键
type RateLimitKey struct {
	Type  string // 限流类型：ip, user, endpoint
	Value string // 限流值：IP地址、用户ID、端点名称
}

// String 返回字符串表示
func (k RateLimitKey) String() string {
	return fmt.Sprintf("%s:%s", k.Type, k.Value)
}

// NewIPKey 创建IP限流键
func NewIPKey(ip string) RateLimitKey {
	return RateLimitKey{Type: "ip", Value: ip}
}

// NewUserKey 创建用户限流键
func NewUserKey(userID string) RateLimitKey {
	return RateLimitKey{Type: "user", Value: userID}
}

// NewEndpointKey 创建端点限流键
func NewEndpointKey(endpoint string) RateLimitKey {
	return RateLimitKey{Type: "endpoint", Value: endpoint}
}

// RateLimitInfo 限流信息
type RateLimitInfo struct {
	Allowed   bool          `json:"allowed"`
	Limit     int           `json:"limit"`
	Remaining int           `json:"remaining"`
	ResetTime time.Time     `json:"reset_time"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// GetRateLimitInfo 获取限流信息
func (r *RedisRateLimiter) GetRateLimitInfo(ctx context.Context, key string) (*RateLimitInfo, error) {
	now := time.Now()
	window := now.Truncate(r.window)
	redisKey := fmt.Sprintf("rate_limit:%s:%d", key, window.Unix())
	
	currentStr, err := r.client.Get(ctx, redisKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	
	current := 0
	if currentStr != "" {
		current, _ = strconv.Atoi(currentStr)
	}
	
	limit := int(r.defaultLimit) * int(r.window.Seconds())
	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}
	
	resetTime := window.Add(r.window)
	
	info := &RateLimitInfo{
		Allowed:   current < limit,
		Limit:     limit,
		Remaining: remaining,
		ResetTime: resetTime,
	}
	
	if !info.Allowed {
		info.RetryAfter = resetTime.Sub(now)
	}
	
	return info, nil
}