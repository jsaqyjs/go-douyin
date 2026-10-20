package routers

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/middleware"
	"go-douyin/service/feed_service"
	"go-douyin/service/follow_service"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

func invalidateUserFeed(userID uint64) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()
	if err := feed_service.InvalidateFollowingFeed(
		ctx, userID,
	); err != nil {
		slog.Warn(
			"关注关系更新成功，但Feed缓存失效失败",
			"user_id", userID,
			"error", err,
		)
	}
}

// 获取用户ID
func parseFollowUserID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)

	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400, "message": "用户ID不合法",
		})
		return 0, false
	}

	return id, true
}

// 获取分页
func parseFollowPage(c *gin.Context) (int, int, bool) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(400, gin.H{
			"code": 400, "message": "page不合法",
		})
		return 0, 0, false
	}

	size, err := strconv.Atoi(
		c.DefaultQuery("page_size", "10"),
	)
	if err != nil || size < 1 || size > 50 {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "page_size必须在1到50之间",
		})
		return 0, 0, false
	}

	return page, size, true
}

// 统一处理关注业务错误
func handleFollowError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, follow_service.ErrUserNotFound):
		c.JSON(404, gin.H{
			"code": 404, "message": "用户不存在",
		})
	case errors.Is(err, follow_service.ErrFollowSelf):
		c.JSON(400, gin.H{
			"code": 400, "message": "不能关注自己",
		})
	case errors.Is(err, follow_service.ErrInvalidPage):
		c.JSON(400, gin.H{
			"code": 400, "message": "分页参数不合法",
		})
	default:
		slog.Error("关注模块错误", "error", err)
		c.JSON(500, gin.H{
			"code": 500, "message": "服务器内部错误" + err.Error(),
		})
	}

	return true
}

// POST /users/:id/follow
func FollowUserHandler(c *gin.Context) {
	// 获取关注ID
	targetID, ok := parseFollowUserID(c)
	if !ok {
		return
	}
	// 获取自己ID
	currentID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(401, gin.H{
			"code": 401, "message": "未登录",
		})
		return
	}
	changed, err := follow_service.FollowUser(
		currentID, targetID,
	)
	if handleFollowError(c, err) {
		return
	}
	if changed {
		invalidateUserFeed(currentID)
	}
	message := "关注成功"
	if !changed {
		message = "已经关注过该用户"
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": message,
		"data":    gin.H{"is_following": true},
	})
}

// DELETE /users/:id/follow
func UnfollowUserHandler(c *gin.Context) {
	targetID, ok := parseFollowUserID(c)
	if !ok {
		return
	}

	currentID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(401, gin.H{
			"code": 401, "message": "未登录",
		})
		return
	}

	changed, err := follow_service.UnfollowUser(
		currentID, targetID,
	)
	if handleFollowError(c, err) {
		return
	}
	if changed {
		invalidateUserFeed(currentID)
	}
	message := "取消关注成功"
	if !changed {
		message = "当前未关注该用户"
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": message,
		"data":    gin.H{"is_following": false},
	})
}

// GET /users/:id/follow-status
func FollowStatusHandler(c *gin.Context) {
	targetID, ok := parseFollowUserID(c)
	if !ok {
		return
	}

	currentID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(401, gin.H{
			"code": 401, "message": "未登录",
		})
		return
	}

	isFollowing, err := follow_service.IsFollowing(
		currentID, targetID,
	)
	if handleFollowError(c, err) {
		return
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "查询成功",
		"data": gin.H{
			"is_following": isFollowing,
		},
	})
}

// GET /users/:id/following
func ListFollowingHandler(c *gin.Context) {
	userID, ok := parseFollowUserID(c)
	if !ok {
		return
	}

	page, size, ok := parseFollowPage(c)
	if !ok {
		return
	}
	result, err := follow_service.ListFollowing(
		userID, page, size,
	)
	if handleFollowError(c, err) {
		return
	}

	c.JSON(200, gin.H{
		"code": 200, "message": "查询成功",
		"data": result,
	})
}

// GET /users/:id/followers
func ListFollowersHandler(c *gin.Context) {
	userID, ok := parseFollowUserID(c)
	if !ok {
		return
	}

	page, size, ok := parseFollowPage(c)
	if !ok {
		return
	}

	result, err := follow_service.ListFollowers(
		userID, page, size,
	)
	if handleFollowError(c, err) {
		return
	}

	c.JSON(200, gin.H{
		"code": 200, "message": "查询成功",
		"data": result,
	})
}
