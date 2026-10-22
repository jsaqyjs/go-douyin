package routers

import (
	"github.com/gin-gonic/gin"
	"go-douyin/middleware"
	"time"
)

func InitRouter() *gin.Engine {

	r := gin.Default()
	loginLimiter := middleware.NewRedisRateLimiter("douyin:ratelimit:login",
		3,
		time.Minute)
	commentLimiter := middleware.NewRedisRateLimiter("douyin:ratelimit:comment",
		5,
		time.Minute)
	feedConcurrency := middleware.ConcurrencyLimit(20)

	// 不需要登录的接口
	r.POST("/users/register", RegisterHandler)
	r.POST("/users/login", middleware.RedisIPLimit(loginLimiter), LoginHandler)
	// 第44课：公开视频搜索
	r.GET("/videos/search", SearchVideosHandler)
	
	// 公开接口：任何人都能查看正常发布的视频

	r.GET("/videos", ListVideosHandler)
	r.GET("/videos/:id", GetVideoHandler)
	r.GET("/videos/:id/comments", middleware.RedisUserLimit(commentLimiter), ListCommentsHandler)
	r.GET("/comments/:id/replies", ListRepliesHandler)
	// 用户主页
	r.GET("/users/:id/profile", GetUserProfileHandler)
	// 用户作品列表
	r.GET("/users/:id/videos", GetUserVideosHandler)
	// 公开：关注列表、粉丝列表
	r.GET("/users/:id/following", ListFollowingHandler)
	r.GET("/users/:id/followers", ListFollowersHandler)
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
	// 需要登录：关注、取消关注、关注状态
	auth.POST("/users/:id/follow", FollowUserHandler)
	auth.DELETE("/users/:id/follow", UnfollowUserHandler)
	auth.GET("/users/:id/follow-status", FollowStatusHandler)
	// 关注Feed
	auth.GET("/feed/following", feedConcurrency, FollowingFeedHandler)

	// 第42课：视频收藏
	auth.POST("/videos/:id/favorite", FavoriteVideoHandler)

	auth.DELETE("/videos/:id/favorite", UnfavoriteVideoHandler)

	auth.GET("/users/me/favorites", ListFavoritesHandler)
	return r
}
