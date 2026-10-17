package routers

import (
	"github.com/gin-gonic/gin"
	"go-douyin/middleware"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	// 不需要登录的接口
	r.POST("/users/register", RegisterHandler)
	r.POST("/users/login", LoginHandler)
	// 需要JWT认证的接口组
	auth := r.Group("/")
	auth.Use(middleware.JWTAuthMiddleware())

	auth.GET("/users/me", GetMeHandler)

	return r
}
