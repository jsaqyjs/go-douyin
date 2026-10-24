package event_service

import (
	"context"
	"go-douyin/database"
	"go-douyin/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// ProcessLikeEvent 处理一次点赞业务事件。
// 已成功处理的EventID不会重复执行副作用。
func ProcessLikeEvent(
	ctx context.Context,
	event LikeEvent,
) error {
	if event.EventID == "" ||
		len(event.EventID) > 64 ||
		event.UserID == 0 ||
		event.VideoID == 0 {
		return ErrInvalidEvent
	}

	return database.DB.WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			// 1. 用唯一主键抢占事件处理权。
			processed := models.ProcessedEvent{
				EventID:     event.EventID,
				EventType:   "video.liked",
				ProcessedAt: time.Now(),
			}
			result := tx.Clauses(
				clause.OnConflict{
					DoNothing: true,
				},
			).Create(&processed)
			if result.Error != nil {
				return result.Error
			}

			// 已存在，说明事件之前已经提交过。
			if result.RowsAffected == 0 {
				return nil
			}
			// 2. 真正的业务副作用。
			audit := models.LikeAudit{
				EventID: event.EventID,
				VideoID: event.VideoID,
				UserID:  event.UserID,
			}

			if err := tx.Create(&audit).Error; err != nil {
				return err
			}
			// 3. 返回nil后，GORM会尝试提交事务。
			return nil

		})
}
