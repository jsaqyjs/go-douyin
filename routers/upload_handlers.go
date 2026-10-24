package routers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go-douyin/database"
	"go-douyin/middleware"
	"go-douyin/models"
	"go-douyin/service/upload_service"
	"go-douyin/storage"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func UploadVideoHandler(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请选择视频文件，表单字段必须是file",
		})
		return
	}

	if file.Size > upload_service.MaxVideoSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"code":    413,
			"message": "视频文件不能超过100MiB",
		})
		return
	}
	result, err := upload_service.UploadVideo(
		c.Request.Context(), userID, file,
	)
	if errors.Is(err, upload_service.ErrVideoTooLarge) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"code":    413,
			"message": "视频文件过大",
		})
		return
	}
	if errors.Is(err, upload_service.ErrInvalidVideo) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "仅支持有效的MP4文件",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "视频上传失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

// GetVideoPlayURLHandler 为已发布的视频生成临时播放地址。
func GetVideoPlayURLHandler(c *gin.Context) {
	videoID, err := strconv.ParseUint(
		c.Param("id"), 10, 64,
	)
	if err != nil || videoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "视频ID不合法",
		})
		return
	}
	var video models.Video
	err = database.DB.WithContext(c.Request.Context()).
		Where("id = ? AND status = ?", videoID, 1).
		First(&video).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "视频不存在或已下架",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询视频失败",
		})
		return
	}

	// 新上传视频在数据库中使用 minio:// 标记。
	const prefix = "minio://"
	if len(video.PlayURL) < len(prefix) ||
		video.PlayURL[:len(prefix)] != prefix {
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "该视频不是MinIO存储格式",
		})
		return
	}
	objectKey := video.PlayURL[len(prefix):]
	if storage.MinIO == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "对象存储暂不可用",
		})
		return
	}

	url, err := storage.MinIO.PresignReadURL(
		c.Request.Context(), objectKey,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成播放地址失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"play_url":   url,
			"expires_in": 900,
		},
	})
}
func PublishUploadedVideoHandler(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	var input upload_service.PublishUploadedInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求JSON不合法",
		})
		return
	}

	videoID, err := upload_service.PublishUploadedVideo(
		c.Request.Context(),
		userID,
		input,
	)

	switch {
	case errors.Is(err, upload_service.ErrBadPublish):
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "发布参数不合法",
		})

	case errors.Is(err, upload_service.ErrAssetNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "上传记录不存在",
		})

	case errors.Is(err, upload_service.ErrAssetForbidden):
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "不能发布其他用户的文件",
		})

	case errors.Is(err, upload_service.ErrAssetUnavailable):
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "文件不可发布或已被使用",
		})

	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "发布失败",
		})

	default:
		c.JSON(http.StatusCreated, gin.H{
			"code": 0,
			"data": gin.H{
				"video_id": videoID,
			},
		})
	}
}
