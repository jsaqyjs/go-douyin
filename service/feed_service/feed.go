package feed_service

import (
	"context"
	"errors"
	"fmt"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"go-douyin/service/video_service"
	"go-douyin/utils"
)

var ErrInvalidLimit = errors.New("limit必须在1到50之间")

// ListFollowingFeed 查询关注用户发布的视频
func ListFollowingFeed(
	ctx context.Context,
	currentUserID uint64,
	limit int,
	cursorString string,
) (dto.FollowingFeedResponse, error) {
	result := dto.FollowingFeedResponse{
		List: make([]dto.VideoResponse, 0),
	}
	// 1. 校验分页大小
	if limit < 1 || limit > 50 {
		return result, ErrInvalidLimit
	}
	// 2. 创建查询
	// follows 与 videos 通过作者ID关联
	query := database.DB.WithContext(ctx).
		Model(&models.Video{}).
		Joins("join follow as f on "+
			"f.following_id = video.author_id ").
		Where("f.follower_id = ?", currentUserID).
		Where("video.status = ?", models.VideoStatusPublished)
	// 3. 如果有游标，则只查询比游标更旧的视频
	if cursorString != "" {
		cursor, err := utils.DecodeFeedCursor(cursorString)
		if err != nil {
			return result, fmt.Errorf(
				"游标不合法: %w", err,
			)
		}
		query = query.Where(
			`(video.created_at < ? ) or (video.created_at = ? and  video.id < ?)`,
			cursor.Time,
			cursor.Time,
			cursor.ID)
	}
	// 4. 多查询一条，判断是否还有下一页
	var videos []models.Video
	err := query.
		Select("video.*").
		Order("video.created_at DESC").
		Order("video.id DESC").
		Limit(limit + 1).
		Find(&videos).Error
	if err != nil {
		return result, fmt.Errorf(
			"查询关注Feed失败: %w", err,
		)
	}
	// 5. 判断是否存在下一页
	if len(videos) > limit {
		result.HasMore = true
		videos = videos[:limit]
	}
	// 6. 把数据库模型转换成视频DTO
	for _, video := range videos {
		result.List = append(
			result.List,
			video_service.ToVideoResponse(video),
		)
	}
	// 7. 有下一页时生成下一页游标
	if result.HasMore && len(videos) > 0 {
		lastVideo := videos[len(videos)-1]
		nextCursor, err := utils.EncodeFeedCursor(
			lastVideo.CreatedAt,
			lastVideo.ID,
		)

		if err != nil {
			return result, err
		}

		result.NextCursor = nextCursor
	}

	return result, nil
}
