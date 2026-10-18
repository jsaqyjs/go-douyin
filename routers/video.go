package routers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/dto"
	"go-douyin/middleware"
	"go-douyin/service/video_service"
	"log/slog"
	"net/http"
	"strconv"
)

// POST /videos
func PublishVideoHandler(c *gin.Context) {
	var req dto.PublishVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "视频参数不合法",
			"data":    nil,
		})
		return
	}
	// 获取 JWT 中的当前用户ID
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
			"data":    nil,
		})
		return
	}

	video, queued, err := video_service.PublishAndEnqueueVideo(userID, req)
	if err != nil {
		slog.Error("发布视频失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "发布视频失败",
			"data":    nil,
		})
		return
	}
	if !queued {
		c.JSON(http.StatusAccepted, gin.H{
			"code":    202,
			"message": "视频已保存，Feed分发待确认",
			"data": gin.H{
				"video":       video,
				"feed_queued": false,
			},
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "发布成功",
		"data": gin.H{
			"video":       video,
			"feed_queued": true,
		},
	})
}

// GET /videos/:id
func GetVideoHandler(c *gin.Context) {
	// 路由参数原本是字符串
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
		return

	}
	video, err := video_service.GetVideoByID(videoID)
	if errors.Is(err, video_service.ErrVideoNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "视频不存在",
			"data":    nil,
		})
		return
	}

	if err != nil {
		slog.Error("查询视频失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询视频失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data":    video,
	})
}

// GET /videos?page=1&page_size=10
func ListVideosHandler(c *gin.Context) {
	page, err := strconv.Atoi(
		c.DefaultQuery("page", "1"),
	)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "page参数不合法",
			"data":    nil,
		})
		return
	}

	pageSize, err := strconv.Atoi(
		c.DefaultQuery("page_size", "10"),
	)
	if err != nil || pageSize < 1 || pageSize > 50 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "page_size必须在1到50之间",
			"data":    nil,
		})
		return
	}
	result, err := video_service.ListVideos(page, pageSize)
	if err != nil {
		if err.Error() == "分页参数不合法" ||
			err.Error() == "分页偏移量过大" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}

		slog.Error("查询视频列表失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询视频列表失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data":    result,
	})
}
