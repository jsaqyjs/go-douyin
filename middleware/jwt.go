package middleware

import (
	"log"
	"net/http"
	"strings"

	"go-douyin/utils"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware JWT身份认证中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 1. 获取 Authorization 请求头
		authHeader := c.GetHeader("Authorization")

		// 2. 要求 Bearer Token 格式
		parts := strings.Fields(authHeader)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {

			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"code":    401,
					"message": "缺少或无效的Authorization",
					"data":    nil,
				},
			)
			return
		}

		// 3. 解析 JWT，获得当前用户 ID
		userID, err := utils.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"code":    401,
					"message": "Token无效或已过期",
					"data":    nil,
				},
			)
			return
		}

		// 4. 将用户ID保存到当前请求的Context中
		c.Set("userID", userID)

		// 5. 继续执行后面的Handler
		c.Next()
	}
}

// GetCurrentUserID 获取当前请求中的用户ID
func GetCurrentUserID(c *gin.Context) (uint64, bool) {
	value, exists := c.Get("userID")

	if !exists {
		return 0, false
	}

	userID, ok := value.(uint64)
	log.Printf("解析 value 和userID  %v，%v", value, userID)
	return userID, ok && userID != 0
}
