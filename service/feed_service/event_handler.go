package feed_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-douyin/database"
	"go-douyin/models"
	"go-douyin/mq"
)

var ErrPermanent = errors.New("永久错误")

func HandlePublishedEvent(
	ctx context.Context,
	body []byte, // json 数据
) error {
	var event mq.VideoPublishedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf(
			"%w: JSON解析失败: %v",
			ErrPermanent, err,
		)
	}

	if event.VideoID == 0 ||
		event.AuthorID == 0 ||
		event.CreatedAt.IsZero() {
		return fmt.Errorf(
			"%w: 消息字段缺失",
			ErrPermanent,
		)
	}
	// 以MySQL中的视频为权威数据进行校验
	var video models.Video

	err := database.DB.WithContext(ctx).
		Where("id = ?", event.VideoID).
		First(&video).Error

	if err != nil {
		// 数据库故障或视频暂时查不到，
		// 先交由有次数限制的重试机制处理。
		return fmt.Errorf("查询视频失败: %w", err)
	}

	if video.AuthorID != event.AuthorID {
		return fmt.Errorf(
			"%w: 作者ID与视频记录不匹配",
			ErrPermanent,
		)
	}

	if video.Status != models.VideoStatusPublished {
		// 已下架视频不再分发。
		return nil
	}

	return FanoutVideo(
		ctx,
		video.AuthorID,
		video.ID,
		video.CreatedAt,
	)
}
