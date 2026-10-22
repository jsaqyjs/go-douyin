package profile_service

import (
	"context"
	"errors"
	"fmt"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"go-douyin/utils"
	"gorm.io/gorm"
	"time"
)

var ErrUserNotFound = errors.New("用户不存在")

// GetUserProfile 查询用户主页
func GetUserProfile(
	ctx context.Context,
	userID uint64,
) (dto.UserProfileResponse, error) {
	var result dto.UserProfileResponse
	db := database.DB.WithContext(ctx)

	// 1. 查询用户基本信息
	var user models.User

	err := db.First(&user, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, ErrUserNotFound
	}

	if err != nil {
		return result, err
	}
	result.ID = user.ID
	result.Username = user.Username
	result.Nickname = user.Nickname
	result.AvatarURL = user.AvatarURL

	// 2. 查询关注数量
	err = db.Model(&models.Follow{}).
		Where("follower_id = ?", userID).
		Count(&result.FollowingCount).Error
	if err != nil {
		return result, fmt.Errorf(
			"查询关注数失败: %w", err,
		)
	}

	// 3. 查询粉丝数量
	err = db.Model(&models.Follow{}).
		Where("following_id = ?", userID).
		Count(&result.FollowerCount).Error

	if err != nil {
		return result, fmt.Errorf(
			"查询粉丝数失败: %w", err,
		)
	}

	// 4. 查询公开作品数量
	// 4. 查询公开作品数量
	err = db.Model(&models.Video{}).
		Where(
			"author_id = ? AND status = ?",
			userID, 1,
		).
		Count(&result.VideoCount).Error

	if err != nil {
		return result, fmt.Errorf(
			"查询作品数失败: %w", err,
		)
	}

	// 5. 统计该用户公开视频的总获赞量
	err = db.Model(&models.Video{}).
		Select("COALESCE(SUM(like_count), 0)").
		Where(
			"author_id = ? AND status = ?",
			userID, 1,
		).
		Scan(&result.TotalLikes).Error
	if err != nil {
		return result, fmt.Errorf(
			"查询获赞数失败: %w", err,
		)
	}

	return result, nil
}

// NormalizePagination 规范分页参数。
// page 从1开始，limit最大50。
func NormalizePagination(
	page int,
	limit int,
) (int, int) {
	if page < 1 {
		page = 1
	}

	if page > 10000 {
		page = 10000
	}

	if limit < 1 {
		limit = 10
	}

	if limit > 50 {
		limit = 50
	}

	return page, limit
}

// ListUserVideos 查询指定用户发布的公开视频
func ListUserVideos(
	ctx context.Context,
	userID uint64,
	limit int,
	cursorString string,
) (dto.UserVideosResponse, error) {
	result := dto.UserVideosResponse{
		List: make([]dto.ProfileVideoResponse, 0),
	}
	// 默认10条，最多50条
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	// 解析游标。第一页没有游标
	var cursor utils.FeedCursor
	if cursorString != "" {
		var err error
		cursor, err = utils.DecodeFeedCursor(cursorString)
		if err != nil {
			return result, err
		}
	}
	db := database.DB.WithContext(ctx)
	// 确认用户存在
	var user models.User
	err := db.Select("id").
		First(&user, userID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, ErrUserNotFound
	}
	if err != nil {
		return result, err
	}

	if err != nil {
		return result, err
	}
	// 基础查询：只查询该用户的公开视频
	query := db.Model(&models.Video{}).
		Where(
			"author_id = ? AND status = ?",
			userID, 1,
		)
	if cursorString != "" {
		query = query.Where(
			`(created_at < ? OR
              (created_at = ? AND id < ?))`,
			cursor.Time,
			cursor.Time,
			cursor.ID,
		)
	}
	// 多查一条，判断是否有下一页
	var videos []models.Video
	err = query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit + 1).
		Find(&videos).Error

	if err != nil {
		return result, err
	}

	result.HasMore = len(videos) > limit

	if result.HasMore {
		videos = videos[:limit]
	}
	// 将数据库模型转换为API响应
	for _, video := range videos {
		result.List = append(
			result.List,
			dto.ProfileVideoResponse{
				ID:           video.ID,
				Title:        video.Title,
				Description:  video.Description,
				PlayURL:      video.PlayURL,
				CoverURL:     video.CoverURL,
				LikeCount:    video.LikeCount,
				CommentCount: video.CommentCount,
				CreatedAt: video.CreatedAt.Format(
					time.RFC3339Nano,
				),
			},
		)

	}
	// 只要还有下一页，就使用当前页最后一条
	// 视频的位置生成下一次请求的游标
	if result.HasMore && len(videos) > 0 {
		last := videos[len(videos)-1]

		result.NextCursor, err = utils.EncodeFeedCursor(
			last.CreatedAt,
			last.ID,
		)
		if err != nil {
			return dto.UserVideosResponse{}, err
		}
	}

	return result, nil
}
