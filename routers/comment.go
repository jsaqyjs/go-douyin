package routers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/dto"
	"go-douyin/middleware"
	"go-douyin/service/comment_service"
	"log/slog"
	"net/http"
	"strconv"
)

// parseCommentID 获取video id 类型的转换
func parseCommentID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400, "message": "ID不合法", "data": nil,
		})
		return 0, false
	}
	return id, true
}

// 限制分页，分页大小
func parseCommentPage(c *gin.Context) (int, int, bool) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(400, gin.H{
			"code": 400, "message": "page不合法",
		})
		return 0, 0, false
	}

	size, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || size < 1 || size > 50 {
		c.JSON(400, gin.H{
			"code": 400, "message": "page_size必须在1到50之间",
		})
		return 0, 0, false
	}

	return page, size, true
}

// POST /videos/:id/comments
func CreateCommentHandler(c *gin.Context) {
	videoID, ok := parseCommentID(c, "id")
	if !ok {
		return
	}

	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(401, gin.H{"code": 401, "message": "未登录"})
		return
	}

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code": 400, "message": "评论参数不合法",
		})
		return
	}

	result, err := comment_service.CreateComment(videoID, userID, req)
	if errors.Is(err, comment_service.ErrVideoNotFound) {
		c.JSON(404, gin.H{
			"code": 404, "message": "视频不存在或不可评论",
		})
		return
	}
	if errors.Is(err, comment_service.ErrInvalidParent) ||
		errors.Is(err, comment_service.ErrInvalidContent) {
		c.JSON(400, gin.H{
			"code": 400, "message": err.Error(),
		})
		return
	}
	if err != nil {
		slog.Error("发表评论失败", "error", err)
		c.JSON(500, gin.H{
			"code": 500, "message": "发表评论失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 201, "message": "评论成功", "data": result,
	})
}

// GET /videos/:id/comments
func ListCommentsHandler(c *gin.Context) {
	videoID, ok := parseCommentID(c, "id")
	if !ok {
		return
	}

	page, size, ok := parseCommentPage(c)
	if !ok {
		return
	}
	result, err := comment_service.ListComments(
		videoID, page, size,
	)
	sendCommentList(c, result, err)
}

// GET /comments/:id/replies
func ListRepliesHandler(c *gin.Context) {
	parentID, ok := parseCommentID(c, "id")
	if !ok {
		return
	}

	page, size, ok := parseCommentPage(c)
	if !ok {
		return
	}

	result, err := comment_service.ListReplies(
		parentID, page, size,
	)
	sendCommentList(c, result, err)
}

// DELETE /comments/:id
func DeleteCommentHandler(c *gin.Context) {
	commentID, ok := parseCommentID(c, "id")
	if !ok {
		return
	}

	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(401, gin.H{"code": 401, "message": "未登录"})
		return
	}

	err := comment_service.DeleteComment(userID, commentID)

	if errors.Is(err, comment_service.ErrCommentNotFound) {
		c.JSON(404, gin.H{
			"code": 404, "message": "评论不存在",
		})
		return
	}
	if errors.Is(err, comment_service.ErrForbidden) {
		c.JSON(403, gin.H{
			"code": 403, "message": "无权删除其他用户的评论",
		})
		return
	}
	if err != nil {
		slog.Error("删除评论失败", "error", err)
		c.JSON(500, gin.H{
			"code": 500, "message": "删除评论失败",
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 200, "message": "删除成功",
	})
}
func sendCommentList(
	c *gin.Context,
	result dto.CommentListResponse,
	err error,
) {
	if errors.Is(err, comment_service.ErrVideoNotFound) ||
		errors.Is(err, comment_service.ErrCommentNotFound) {
		c.JSON(404, gin.H{
			"code": 404, "message": "视频或评论不存在",
		})
		return
	}
	if errors.Is(err, comment_service.ErrInvalidPage) {
		c.JSON(400, gin.H{
			"code": 400, "message": "分页参数不合法",
		})
		return
	}
	if err != nil {
		slog.Error("获取评论列表失败", "error", err)
		c.JSON(500, gin.H{
			"code": 500, "message": "查询评论失败",
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 200, "message": "查询成功", "data": result,
	})
}
