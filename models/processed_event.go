package models

import "time"

type ProcessedEvent struct {
	EventID     string    `gorm:"type:varchar(64);primaryKey"`
	EventType   string    `gorm:"type:varchar(64);not null"`
	ProcessedAt time.Time `gorm:"not null"`
}

func (ProcessedEvent) TableName() string {
	return "processed_event"
}

//{
//  "event_id": "like-10001",
//  "video_id": 15,
//  "user_id": 3
//}
