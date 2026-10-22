package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// 限流（Rate Limiting）
// 保存每个用户/IP自己的限流器
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitStore 管理多个限流器
type RateLimitStore struct {
	mu       sync.Mutex
	visitors map[string]*visitor

	rate  rate.Limit
	burst int
}

func NewRateLimitStore(
	requestsPerMinute int,
	burst int,
) *RateLimitStore {
	if requestsPerMinute <= 0 || burst <= 0 {
		panic("限流参数必须大于0")
	}

	return &RateLimitStore{
		visitors: make(map[string]*visitor),
		rate: rate.Limit(
			float64(requestsPerMinute) / 60.0,
		),
		burst: burst,
	}
}

// Allow 判断该Key是否允许一次请求
func (s *RateLimitStore) Allow(key string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	v, exists := s.visitors[key]
	if !exists {
		v = &visitor{
			limiter: rate.NewLimiter(s.rate, s.burst),
		}
		s.visitors[key] = v
	}
	v.lastSeen = now
	return v.limiter.Allow()
}

// Cleanup 删除长期不活跃的客户端记录
func (s *RateLimitStore) Cleanup(idle time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, v := range s.visitors {
		// > idle（idle 代表空闲、闲置的意思）：如果过去了很久（比如超过 10 分钟），说明这人早就不玩了。
		// now.Sub(...) 就是算一下：“距离他上次来，过去了多久？”
		if now.Sub(v.lastSeen) > idle {
			delete(s.visitors, key)
		}
	}
}

// IPLimit 根据客户端IP限流
func IPLimit(store *RateLimitStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ip:" + c.ClientIP()

		if !store.Allow(key) {
			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				gin.H{
					"code":    429,
					"message": "请求过于频繁，请稍后重试",
				},
			)
			return
		}

		c.Next()
	}
}

// UserLimit 根据JWT用户ID限流
// 必须放在JWTAuthMiddleware之后
func UserLimit(store *RateLimitStore) gin.HandlerFunc {
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

		key := "user:" + strconv.FormatUint(userID, 10)

		if !store.Allow(key) {
			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				gin.H{
					"code":    429,
					"message": "操作太频繁，请稍后再试",
				},
			)
			return
		}

		c.Next()
	}
}

// StartCleanup 定期清除不活跃的限流器。
// ctx取消时自动停止。
func (s *RateLimitStore) StartCleanup(
	ctx context.Context,
	interval time.Duration,
	idle time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.Cleanup(idle)
		}
	}
}
