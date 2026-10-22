package routers

import (
	"github.com/gin-gonic/gin"
	"go-douyin/service/hot_service"
	"net/http"
	"strconv"
)

func ListHotVideosHandler(c *gin.Context) {
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

	result, err := hot_service.ListHotVideos(
		c.Request.Context(),
		limit,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取热门视频失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}
