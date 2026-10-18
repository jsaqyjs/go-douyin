package outbox_service

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-douyin/database"
	"go-douyin/models"
	"go-douyin/mq"
	"gorm.io/gorm"
	"log/slog"
	"time"
)

// DispatchOnce 尝试投递一条待发送事件
// 返回值 bool 表示是否找到并处理了一条事件
func DispatchOnce() (bool, error) {
	var event models.OutboxEvent
	// 1. 查询最早的一条待发送事件
	err := database.DB.
		Where("status = ?", models.OutboxPending).
		Order("id ASC").
		First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("查询Outbox失败: %w", err)
	}
	// 2. 当前课程只处理视频发布事件
	if event.EventType != mq.RoutingKey {
		return true, fmt.Errorf(
			"不支持的事件类型: %s",
			event.EventType,
		)
	}
	// 3. 反序列化消息
	var payload mq.VideoPublishedEvent
	if err := json.Unmarshal(
		[]byte(event.Payload),
		&payload,
	); err != nil {
		return true, fmt.Errorf("Outbox JSON不合法: %w", err)
	}
	// 4. 发送RabbitMQ消息
	publishErr := mq.PublishVideoPublished(payload)
	if publishErr != nil {
		if err := database.DB.Model(&models.OutboxEvent{}).
			Where(
				"id = ? AND status = ?",
				event.ID,
				models.OutboxPending,
			).
			UpdateColumn(
				"retry_count",
				gorm.Expr("retry_count + 1"),
			).Error; err != nil {
			return true, fmt.Errorf(
				"发布失败且更新重试次数失败: %w", err,
			)

		}
		return true, fmt.Errorf(
			"RabbitMQ投递失败: %w", publishErr,
		)

	}
	// 5. Broker确认成功，标记已发送
	now := time.Now()

	err = database.DB.Model(&models.OutboxEvent{}).
		Where(
			"id = ? AND status = ?",
			event.ID,
			models.OutboxPending,
		).
		Updates(map[string]any{
			"status":  models.OutboxSent,
			"sent_at": now,
		}).Error

	if err != nil {
		// RabbitMQ可能已收到消息，但状态未能写回MySQL
		// 下次执行会再次投递，因此Consumer必须幂等
		return true, fmt.Errorf(
			"事件已投递但状态更新失败: %w", err,
		)
	}

	slog.Info(
		"Outbox事件发送成功",
		"outbox_id", event.ID,
		"video_id", payload.VideoID,
	)
	return true, nil
}
