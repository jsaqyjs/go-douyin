package routers

import (
	"errors"
	"net/http"
	"strconv"

	"go-douyin/middleware"
	"go-douyin/service/recommend_service"

	"github.com/gin-gonic/gin"
)

// ListRecommendFeedHandler 获取个性化推荐Feed
func ListRecommendFeedHandler(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	limit := 10

	if raw, exists := c.GetQuery("limit"); exists {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 50 {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "limit必须是1到50的整数",
			})
			return
		}
		limit = n
	}

	result, err := recommend_service.ListRecommendFeed(
		c.Request.Context(),
		userID,
		limit,
		c.Query("cursor"),
	)

	if errors.Is(err, recommend_service.ErrInvalidCursor) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "cursor参数不合法",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取推荐视频失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}
