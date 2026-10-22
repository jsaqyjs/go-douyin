package models

import "time"

type VideoFavorite struct {
	ID uint64 `gorm:"primaryKey"`

	UserID uint64 `gorm:"not null;uniqueIndex:uk_user_video,priority:1;index:idx_user_created,priority:1"`

	VideoID uint64 `gorm:"not null;uniqueIndex:uk_user_video,priority:2"`

	CreatedAt time.Time `gorm:"not null;index:idx_user_created,priority:2,sort:desc"`
}

func (VideoFavorite) TableName() string {
	return "video_favorite"
}
