package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestIPLimit 测试IP请求频率限制
func TestIPLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := NewRateLimitStore(2, 2)
	r := gin.New()

	r.GET("/test",
		IPLimit(store),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "ok",
			})
		},
	)

	expected := []int{200, 200, 429}

	for i, want := range expected {
		req := httptest.NewRequest(
			http.MethodGet,
			"/test",
			nil,
		)
		req.RemoteAddr = "127.0.0.1:12345"

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != want {
			t.Fatalf(
				"第%d次请求，期望%d，实际%d",
				i+1, want, w.Code,
			)
		}
	}
}

// TestConcurrencyLimit 测试并发数限制
func TestConcurrencyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	entered := make(chan struct{})
	release := make(chan struct{})

	r := gin.New()

	r.GET("/slow",
		ConcurrencyLimit(1),
		func(c *gin.Context) {
			close(entered) // 第一条请求已经进入业务处理
			<-release      // 等待测试允许它结束

			c.Status(http.StatusOK)
		},
	)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		req := httptest.NewRequest(
			http.MethodGet,
			"/slow",
			nil,
		)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("第一条请求应该成功，实际%d", w.Code)
		}
	}()

	// 确保第一条请求先占用槽位
	<-entered

	// 第二条请求应该被拒绝
	req := httptest.NewRequest(
		http.MethodGet,
		"/slow",
		nil,
	)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf(
			"第二条请求应该返回503，实际%d",
			w.Code,
		)
	}

	// 释放第一条请求
	close(release)
	wg.Wait()
}
