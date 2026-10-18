package models

import "time"

const (
	OutboxPending uint8 = 0
	OutboxSent    uint8 = 1
)

type OutboxEvent struct {
	ID uint64 `gorm:"primaryKey;autoIncrement"`

	// 事件名称，如 video.published
	EventType string `gorm:"size:100;not null"`

	// 消息JSON
	Payload string `gorm:"type:text;not null"`

	// 0=等待发送，1=已发送
	Status uint8 `gorm:"not null;default:0;index:idx_outbox_status_id,priority:1"`

	// 已尝试投递的次数
	RetryCount int `gorm:"not null;default:0"`

	CreatedAt time.Time `gorm:"index:idx_outbox_status_id,priority:2"`
	UpdatedAt time.Time

	SentAt *time.Time
}

func (OutboxEvent) TableName() string {
	return "outbox_event"
}
