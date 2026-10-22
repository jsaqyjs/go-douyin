package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ConcurrencyLimit 限制同时执行的请求数
func ConcurrencyLimit(maxConcurrent int) gin.HandlerFunc {
	if maxConcurrent <= 0 {
		panic("maxConcurrent必须大于0")
	}
	// 带缓冲Channel充当信号量
	sem := make(chan struct{}, maxConcurrent)
	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			// 成功取得一个并发槽位

			defer func() {
				<-sem // 请求结束，释放槽位
			}()

			c.Next()

		default:
			// 所有槽位已满
			c.AbortWithStatusJSON(
				http.StatusServiceUnavailable,
				gin.H{
					"code":    503,
					"message": "服务器繁忙，请稍后重试",
				},
			)
		}
	}
}
