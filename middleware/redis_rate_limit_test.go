package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-douyin/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestRedisSharedRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 使用本地测试Redis
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis未启动: %v", err)
	}

	// 本测试不调用 t.Parallel()
	oldRedis := database.Redis
	database.Redis = client
	defer func() {
		database.Redis = oldRedis
	}()

	// 使用唯一测试Key，避免与正常业务冲突
	prefix := fmt.Sprintf(
		"test:ratelimit:%d",
		time.Now().UnixNano(),
	)

	limiterA := NewRedisRateLimiter(
		prefix, 3, time.Minute,
	)

	limiterB := NewRedisRateLimiter(
		prefix, 3, time.Minute,
	)

	defer client.Del(
		context.Background(),
		prefix+":ip:127.0.0.1",
	)

	// 模拟Gin实例A
	routerA := gin.New()
	routerA.GET("/test", RedisIPLimit(limiterA),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	// 模拟Gin实例B
	routerB := gin.New()
	routerB.GET("/test", RedisIPLimit(limiterB),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	send := func(r *gin.Engine) int {
		req := httptest.NewRequest(
			http.MethodGet, "/test", nil,
		)
		req.RemoteAddr = "127.0.0.1:12345"

		recorder := httptest.NewRecorder()
		r.ServeHTTP(recorder, req)

		return recorder.Code
	}

	// 两个Gin实例交替发送请求
	tests := []struct {
		router *gin.Engine
		want   int
	}{
		{routerA, 200},
		{routerB, 200},
		{routerA, 200},
		{routerB, 429},
	}

	for i, tt := range tests {
		got := send(tt.router)
		if got != tt.want {
			t.Fatalf(
				"第%d次请求: 期望%d, 实际%d",
				i+1, tt.want, got,
			)
		}
	}

	t.Log("两个Gin实例成功共享Redis限流状态")
}
