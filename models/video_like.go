package models

import "time"

type VideoLike struct {
	ID uint64 `gorm:"primaryKey;autoIncrement"`

	UserID    uint64 `gorm:"not null;uniqueIndex:uk_user_video,priority:1"`
	VideoID   uint64 `gorm:"not null;uniqueIndex:uk_user_video,priority:2;index"`
	CreatedAt time.Time
}

func (VideoLike) TableName() string {
	return "video_like"
}
