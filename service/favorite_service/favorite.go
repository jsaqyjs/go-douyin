package favorite_service

import (
	"context"
	"errors"
	"fmt"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

var (
	ErrVideoNotFound = errors.New("视频不存在或不可收藏")
	ErrInvalidUser   = errors.New("用户ID不合法")
	ErrInvalidVideo  = errors.New("视频ID不合法")
)

// FavoriteVideo 收藏视频。
// 重复收藏不会重复插入数据。
func FavoriteVideo(
	ctx context.Context,
	userID uint64,
	videoID uint64,
) error {
	if userID == 0 {
		return ErrInvalidUser
	}
	if videoID == 0 {
		return ErrInvalidVideo
	}

	db := database.DB.WithContext(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		var video models.Video

		// 只允许收藏公开且存在的视频
		err := tx.Select("id").
			Where("id = ? AND status = ?", videoID, 1).
			First(&video).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVideoNotFound
		}
		if err != nil {
			return err
		}

		favorite := models.VideoFavorite{
			UserID:  userID,
			VideoID: videoID,
		}

		// 唯一索引保证数据库不会出现重复收藏
		// DoNothing让重复请求也返回成功。
		err = tx.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&favorite).Error

		return err
	})

}

// UnfavoriteVideo 取消收藏。
// 即使原本没有收藏，也返回成功。
func UnfavoriteVideo(
	ctx context.Context,
	userID uint64,
	videoID uint64,
) error {
	if userID == 0 {
		return ErrInvalidUser
	}
	if videoID == 0 {
		return ErrInvalidVideo
	}

	return database.DB.WithContext(ctx).
		Where(
			"user_id = ? AND video_id = ?",
			userID, videoID,
		).
		Delete(&models.VideoFavorite{}).Error
}

// NormalizeFavoritePagination 规范分页参数
func NormalizeFavoritePagination(
	page, limit int,
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

type favoriteRow struct {
	models.Video
	FavoritedAt time.Time `gorm:"column:favorited_at"`
}

// ListFavorites 查询当前用户的收藏列表
func ListFavorites(
	ctx context.Context,
	userID uint64,
	page int,
	limit int,
) (dto.FavoriteListResponse, error) {
	page, limit = NormalizeFavoritePagination(page, limit)

	result := dto.FavoriteListResponse{
		List:  make([]dto.FavoriteVideoItem, 0),
		Page:  page,
		Limit: limit,
	}

	if userID == 0 {
		return result, ErrInvalidUser
	}

	db := database.DB.WithContext(ctx)
	// 只统计还可以公开访问的视频
	base := db.Table("video_favorite AS f").
		Joins("JOIN video AS v ON v.id = f.video_id").
		Where("f.user_id = ? AND v.status = ?", userID, 1)

	// 查询收藏总数
	if err := base.Count(&result.Total).Error; err != nil {
		return result, fmt.Errorf("统计收藏数失败: %w", err)
	}

	offset := (page - 1) * limit

	var rows []favoriteRow

	err := db.Table("video_favorite AS f").
		Select("v.*, f.created_at AS favorited_at").
		Joins("JOIN video AS v ON v.id = f.video_id").
		Where("f.user_id = ? AND v.status = ?", userID, 1).
		Order("f.created_at DESC").
		Order("f.id DESC").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error

	if err != nil {
		return result, fmt.Errorf("查询收藏列表失败: %w", err)
	}

	for _, row := range rows {
		result.List = append(result.List, dto.FavoriteVideoItem{
			ID:           row.ID,
			AuthorID:     row.AuthorID,
			Title:        row.Title,
			Description:  row.Description,
			PlayURL:      row.PlayURL,
			CoverURL:     row.CoverURL,
			LikeCount:    row.LikeCount,
			CommentCount: row.CommentCount,
			FavoritedAt: row.FavoritedAt.Format(
				time.RFC3339Nano,
			),
		})
	}

	result.HasMore = int64(offset)+
		int64(len(rows)) < result.Total

	return result, nil
}
