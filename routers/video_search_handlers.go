package routers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/service/video_search_service"
	"net/http"
	"strconv"
)

// SearchVideosHandler 视频搜索接口
func SearchVideosHandler(c *gin.Context) {
	keyword := c.Query("keyword")

	limit := 10

	// 解析可选的limit参数
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

	cursor := c.Query("cursor")

	result, err := video_search_service.SearchVideos(
		c.Request.Context(),
		keyword,
		limit,
		cursor,
	)

	if errors.Is(
		err,
		video_search_service.ErrInvalidKeyword,
	) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "搜索关键词必须是1到50个字符",
		})
		return
	}

	if errors.Is(
		err,
		video_search_service.ErrInvalidCursor,
	) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "搜索游标不合法",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "搜索视频失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}
