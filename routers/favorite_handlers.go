package routers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/middleware"
	"go-douyin/service/favorite_service"
	"net/http"
	"strconv"
)

func parseFavoriteVideoID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "视频ID不合法",
		})
		return 0, false
	}
	return id, true
}

// FavoriteVideoHandler 收藏视频
func FavoriteVideoHandler(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	videoID, ok := parseFavoriteVideoID(c)
	if !ok {
		return
	}

	err := favorite_service.FavoriteVideo(
		c.Request.Context(), userID, videoID,
	)

	if errors.Is(err, favorite_service.ErrVideoNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "视频不存在或已下架",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "收藏失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "收藏成功",
	})
}

// UnfavoriteVideoHandler 取消收藏
func UnfavoriteVideoHandler(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	videoID, ok := parseFavoriteVideoID(c)
	if !ok {
		return
	}

	err := favorite_service.UnfavoriteVideo(
		c.Request.Context(), userID, videoID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "取消收藏失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "取消收藏成功",
	})
}

// ListFavoritesHandler 查询我的收藏列表
func ListFavoritesHandler(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	page := 1
	limit := 10

	if raw, exists := c.GetQuery("page"); exists {
		n, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "page必须是整数",
			})
			return
		}
		page = n
	}

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

	result, err := favorite_service.ListFavorites(
		c.Request.Context(), userID, page, limit,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询收藏列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}
