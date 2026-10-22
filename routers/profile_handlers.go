package routers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/service/profile_service"
	"net/http"
	"strconv"
	"strings"
)

// 解析 URL 中的用户ID
func parseProfileUserID(c *gin.Context) (uint64, bool) {
	rawID := c.Param("id")

	userID, err := strconv.ParseUint(rawID, 10, 64)

	if err != nil || userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户ID不合法",
		})
		return 0, false
	}

	return userID, true
}

// GetUserProfileHandler 获取用户主页
func GetUserProfileHandler(c *gin.Context) {
	userID, ok := parseProfileUserID(c)
	if !ok {
		return
	}

	result, err := profile_service.GetUserProfile(
		c.Request.Context(),
		userID,
	)

	if errors.Is(err, profile_service.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "用户不存在",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户主页失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

// GetUserVideosHandler 获取用户作品列表
func GetUserVideosHandler(c *gin.Context) {
	userID, ok := parseProfileUserID(c)
	if !ok {
		return
	}

	limit := 10
	// 3. 获取游标
	cursor := strings.TrimSpace(
		c.Query("cursor"),
	)

	if raw, exists := c.GetQuery("limit"); exists {
		n, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "limit必须是整数",
			})
			return
		}
		limit = n
	}

	result, err := profile_service.ListUserVideos(
		c.Request.Context(),
		userID,
		limit,
		cursor,
	)

	if errors.Is(err, profile_service.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "用户不存在",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询用户作品失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}
