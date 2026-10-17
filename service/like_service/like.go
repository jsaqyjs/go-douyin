package like_service

import (
	"errors"
	"fmt"
	"go-douyin/database"
	"go-douyin/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrVideoNotFound = errors.New("视频不存在或不可点赞")
)

// LikeVideo 给视频点赞
func LikeVideo(userID, videoID uint64) (bool, error) {
	// changed 表示此次请求是否真的新增点赞
	changed := false
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var video models.Video
		// 1. 锁定视频行，保证同一视频的修改串行
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", videoID,
				models.VideoStatusPublished).First(&video).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVideoNotFound
		}
		if err != nil {
			return err
		}
		// 2. 查询当前用户是否已经点赞
		var like models.VideoLike
		err = tx.Where("video_id = ? AND user_id = ?", videoID, userID).First(&like).Error
		// 已点赞：直接返回，不重复计数
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// 3. 新增点赞关系
		newLike := models.VideoLike{
			UserID:  userID,
			VideoID: videoID,
		}
		if err := tx.Create(&newLike).Error; err != nil {
			return err
		}

		// 4. 点赞数量 +1
		if err := tx.Model(&models.Video{}).
			Where("id = ?", videoID).
			UpdateColumn(
				"like_count",
				gorm.Expr("like_count + 1"),
			).Error; err != nil {
			return err
		}

		changed = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("点赞操作失败: %w", err)
	}
	// MySQL事务提交后，清理旧缓存
	if changed {
		DeleteLikeCache(userID, videoID)
	}

	return changed, nil
}

// UnlikeVideo 取消点赞
func UnlikeVideo(userID, videoID uint64) (bool, error) {
	changed := false
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var video models.Video

		// 1. 锁住视频行
		err := tx.Clauses(clause.Locking{
			Strength: "UPDATE",
		}).
			Where("id = ?", videoID).
			First(&video).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVideoNotFound
		}
		if err != nil {
			return err
		}

		// 2. 删除用户点赞关系
		result := tx.Where(
			"user_id = ? AND video_id = ?",
			userID,
			videoID,
		).Delete(&models.VideoLike{})

		if result.Error != nil {
			return result.Error
		}

		// 没有记录：说明本来就没有点赞
		if result.RowsAffected == 0 {
			return nil
		}

		// 3. 点赞数量 -1
		result = tx.Model(&models.Video{}).
			Where("id = ? AND like_count > 0", videoID).
			UpdateColumn(
				"like_count",
				gorm.Expr("like_count - 1"),
			)

		if result.Error != nil {
			return result.Error
		}

		// 正常情况下，计数必须能够扣减
		if result.RowsAffected != 1 {
			return errors.New("点赞计数异常，事务回滚")
		}

		changed = true
		return nil
	})

	if err != nil {
		return false, fmt.Errorf("取消点赞失败: %w", err)
	}
	if changed {
		DeleteLikeCache(userID, videoID)
	}
	return changed, nil
}

// IsVideoLiked 查询当前用户是否点赞
func IsVideoLiked(userID, videoID uint64) (bool, error) {
	// 检查视频是否存在且正常发布
	var video models.Video

	err := database.DB.
		Where(
			"id = ? AND status = ?",
			videoID,
			models.VideoStatusPublished,
		).
		First(&video).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, ErrVideoNotFound
	}
	if err != nil {
		return false, err
	}
	// 2. 查询Redis点赞状态缓存
	if liked, hit := GetLikeCache(userID, videoID); hit {
		return liked, nil
	}

	// 3. Redis未命中，则查询MySQL
	// 统计当前用户对该视频的点赞关系
	var count int64
	err = database.DB.Model(&models.VideoLike{}).
		Where(
			"user_id = ? AND video_id = ?",
			userID,
			videoID,
		).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	liked := count > 0
	// 4. 回填Redis，下次查询可以直接命中
	SetLikeCache(userID, videoID, liked)
	return liked, nil
}
