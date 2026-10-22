package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-douyin/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// 固定窗口限流 Lua：
// 每个Key从第一次请求起计算固定长度的窗口。
// 返回：1=允许，0=拒绝。
var rateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])

if count == 1 then
    redis.call("PEXPIRE", KEYS[1], ARGV[2])
end

if count > tonumber(ARGV[1]) then
    return 0
end

return 1
`)

type RedisRateLimiter struct {
	prefix string
	limit  int64
	window time.Duration
}

func NewRedisRateLimiter(
	prefix string,
	limit int64,
	window time.Duration,
) *RedisRateLimiter {
	if prefix == "" || limit <= 0 || window <= 0 {
		panic("Redis限流参数不合法")
	}

	return &RedisRateLimiter{
		prefix: prefix,
		limit:  limit,
		window: window,
	}
}

// Allow 在Redis中执行原子限流判断
func (l *RedisRateLimiter) Allow(
	ctx context.Context,
	identity string,
) (bool, error) {
	if database.Redis == nil {
		return false, errors.New("Redis尚未初始化")
	}

	key := l.prefix + ":" + identity

	result, err := rateLimitScript.Run(
		ctx,
		database.Redis,
		[]string{key},
		l.limit,
		l.window.Milliseconds(),
	).Int64()

	if err != nil {
		return false, fmt.Errorf("Redis限流失败: %w", err)
	}

	return result == 1, nil
}

// RedisIPLimit 根据客户端IP限流
func RedisIPLimit(
	limiter *RedisRateLimiter,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			300*time.Millisecond,
		)
		defer cancel()

		allowed, err := limiter.Allow(
			ctx,
			"ip:"+c.ClientIP(),
		)

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusServiceUnavailable,
				gin.H{
					"code":    503,
					"message": "限流服务暂时不可用",
				},
			)
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				gin.H{
					"code":    429,
					"message": "登录请求过于频繁",
				},
			)
			return
		}

		c.Next()
	}
}

// RedisUserLimit 根据JWT中的用户ID限流
func RedisUserLimit(
	limiter *RedisRateLimiter,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := GetCurrentUserID(c)
		if !ok {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"code":    401,
					"message": "未登录",
				},
			)
			return
		}

		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			300*time.Millisecond,
		)
		defer cancel()

		identity := "user:" +
			strconv.FormatUint(userID, 10)

		allowed, err := limiter.Allow(ctx, identity)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusServiceUnavailable,
				gin.H{
					"code":    503,
					"message": "限流服务暂时不可用",
				},
			)
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				gin.H{
					"code":    429,
					"message": "评论过于频繁",
				},
			)
			return
		}

		c.Next()
	}
}
