package routers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/middleware"
	"go-douyin/service/feed_service"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// GET /feed/following
func FollowingFeedHandler(c *gin.Context) {
	// 1. 获取当前登录用户ID
	currentUserID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
			"data":    nil,
		})
		return
	}

	// 2. 解析limit
	limit, err := strconv.Atoi(
		c.DefaultQuery("limit", "10"),
	)

	if err != nil || limit < 1 || limit > 50 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "limit必须在1到50之间",
			"data":    nil,
		})
		return
	}

	// 3. 获取游标
	cursor := strings.TrimSpace(
		c.Query("cursor"),
	)
	if len(cursor) > 512 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "cursor过长",
			"data":    nil,
		})
		return
	}

	// 4. 调用Feed Service
	result, err := feed_service.ListFollowingFeed(
		c.Request.Context(),
		currentUserID,
		limit,
		cursor,
	)
	if err != nil {
		if errors.Is(err, feed_service.ErrInvalidLimit) ||
			strings.HasPrefix(err.Error(), "游标不合法:") {

			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "分页参数不合法",
				"data":    nil,
			})
			return
		}

		slog.Error("查询关注Feed失败", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取关注Feed失败",
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
