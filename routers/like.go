package routers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/middleware"
	"go-douyin/service/like_service"
	"log/slog"
	"net/http"
	"strconv"
)

// 从 URL 中获取视频 ID
func parseLikeVideoID(c *gin.Context) (uint64, bool) {
	videoID, err := strconv.ParseUint(
		c.Param("id"),
		10,
		64,
	)

	if err != nil || videoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "视频ID不合法",
			"data":    nil,
		})
		return 0, false
	}

	return videoID, true
}
func LikeVideoHandler(c *gin.Context) {
	videoID, ok := parseLikeVideoID(c)
	if !ok {
		return
	}

	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
			"data":    nil,
		})
		return
	}

	changed, err := like_service.LikeVideo(userID, videoID)

	if errors.Is(err, like_service.ErrVideoNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "视频不存在或不可点赞",
			"data":    nil,
		})
		return
	}

	if err != nil {
		slog.Error("点赞失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "点赞失败",
			"data":    nil,
		})
		return
	}

	message := "点赞成功"
	if !changed {
		message = "已经点赞过，无需重复点赞"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": message,
		"data": gin.H{
			"is_liked": true,
		},
	})
}

// DELETE /videos/:id/like
func UnlikeVideoHandler(c *gin.Context) {
	videoID, ok := parseLikeVideoID(c)
	if !ok {
		return
	}

	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
			"data":    nil,
		})
		return
	}

	changed, err := like_service.UnlikeVideo(userID, videoID)

	if errors.Is(err, like_service.ErrVideoNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "视频不存在",
			"data":    nil,
		})
		return
	}

	if err != nil {
		slog.Error("取消点赞失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "取消点赞失败",
			"data":    nil,
		})
		return
	}

	message := "取消点赞成功"
	if !changed {
		message = "当前没有点赞，无需取消"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": message,
		"data": gin.H{
			"is_liked": false,
		},
	})
}

// GET /videos/:id/like-status
func GetLikeStatusHandler(c *gin.Context) {
	videoID, ok := parseLikeVideoID(c)
	if !ok {
		return
	}

	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
			"data":    nil,
		})
		return
	}

	isLiked, err := like_service.IsVideoLiked(userID, videoID)

	if errors.Is(err, like_service.ErrVideoNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "视频不存在或不可访问",
			"data":    nil,
		})
		return
	}

	if err != nil {
		slog.Error("查询点赞状态失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询点赞状态失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data": gin.H{
			"is_liked": isLiked,
		},
	})
}
