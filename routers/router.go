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
	// 公开接口：任何人都能查看正常发布的视频
	r.GET("/videos", ListVideosHandler)
	r.GET("/videos/:id", GetVideoHandler)
	r.GET("/videos/:id/comments", ListCommentsHandler)
	r.GET("/comments/:id/replies", ListRepliesHandler)
	// 需要JWT认证的接口组
	auth := r.Group("/")
	auth.Use(middleware.JWTAuthMiddleware())
	auth.GET("/users/me", GetMeHandler)
	// 受保护接口：必须登录才能发布视频
	auth.POST("/videos", PublishVideoHandler)
	auth.POST("/videos/:id/like", LikeVideoHandler)
	auth.DELETE("/videos/:id/like", UnlikeVideoHandler)
	auth.GET("/videos/:id/like-status", GetLikeStatusHandler)
	// 需要登录：发布评论、删除评论
	auth.POST("/videos/:id/comments", CreateCommentHandler)
	auth.DELETE("/comments/:id", DeleteCommentHandler)
	return r
}
