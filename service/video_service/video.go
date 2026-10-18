package video_service

import (
	"errors"
	"fmt"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"go-douyin/mq"
	"gorm.io/gorm"
	"log/slog"
)

var ErrVideoNotFound = errors.New("视频不存在")

// 将数据库模型转换成 DTO
func ToVideoResponse(video models.Video) dto.VideoResponse {
	return dto.VideoResponse{
		ID:           video.ID,
		AuthorID:     video.AuthorID,
		Title:        video.Title,
		Description:  video.Description,
		PlayURL:      video.PlayURL,
		CoverURL:     video.CoverURL,
		LikeCount:    video.LikeCount,
		CommentCount: video.CommentCount,
		CreatedAt:    video.CreatedAt,
	}
}

// PublishAndEnqueueVideo 发布视频并通知Feed
// 第二个返回值表示消息是否已获得Broker确认
func PublishAndEnqueueVideo(
	authorID uint64,
	req dto.PublishVideoRequest,
) (dto.VideoResponse, bool, error) {
	// 1. 保存视频
	video, err := PublishVideo(authorID, req)
	if err != nil {
		return dto.VideoResponse{}, false, err
	}
	// 2. 构造发布事件
	event := mq.VideoPublishedEvent{
		VideoID:   video.ID,
		AuthorID:  video.AuthorID,
		CreatedAt: video.CreatedAt,
	}

	// 3. 发布RabbitMQ消息
	if err := mq.PublishVideoPublished(event); err != nil {
		slog.Error(
			"视频已保存，但Feed事件未确认",
			"video_id", video.ID,
			"error", err,
		)

		return video, false, nil
	}

	return video, true, nil
}

func PublishVideo(authorID uint64,
	req dto.PublishVideoRequest) (dto.VideoResponse, error) {
	video := models.Video{
		AuthorID:     authorID,
		Title:        req.Title,
		Description:  req.Description,
		PlayURL:      req.PlayURL,
		CoverURL:     req.CoverURL,
		Status:       models.VideoStatusPublished,
		LikeCount:    0,
		CommentCount: 0,
	}
	if err := database.DB.Create(&video).Error; err != nil {
		return dto.VideoResponse{},
			fmt.Errorf("创建视频失败: %w", err)
	}

	return ToVideoResponse(video), nil
}

func GetVideoByID(videoID uint64) (dto.VideoResponse, error) {
	var video models.Video

	err := database.DB.
		Where(
			"id = ? AND status = ?",
			videoID,
			models.VideoStatusPublished,
		).
		First(&video).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.VideoResponse{}, ErrVideoNotFound
	}

	if err != nil {
		return dto.VideoResponse{},
			fmt.Errorf("查询视频失败: %w", err)
	}

	return ToVideoResponse(video), nil
}
func ListVideos(page int,
	pageSize int) (dto.VideoListResponse, error) {
	result := dto.VideoListResponse{
		List:     make([]dto.VideoResponse, 0),
		Page:     page,
		PageSize: pageSize,
	}
	if page < 1 || pageSize < 1 || pageSize > 50 {
		return result, errors.New("分页参数不合法")
	}
	// 防止偏移量计算溢出
	maxInt := int(^uint(0) >> 1)

	if page-1 > maxInt/pageSize {
		return result, errors.New("分页偏移量过大")
	}
	offset := (page - 1) * pageSize
	query := database.DB.Model(&models.Video{}).
		Where("status = ?", models.VideoStatusPublished)
	// 查询总记录数
	if err := query.Count(&result.Total).Error; err != nil {
		return result, fmt.Errorf("统计视频数量失败: %w", err)
	}

	var videos []models.Video

	// 查询当前页
	err := query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&videos).Error
	if err != nil {
		return result, fmt.Errorf("查询视频列表失败: %w", err)
	}
	for _, video := range videos {
		result.List = append(
			result.List,
			ToVideoResponse(video),
		)
	}
	return result, nil
}
