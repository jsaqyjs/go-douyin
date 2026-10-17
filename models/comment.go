package models

import "time"

type Comment struct {
	ID uint64 `gorm:"primaryKey;autoIncrement"`

	VideoID uint64 `gorm:"not null;index:idx_comment_video_created,priority:1"`
	UserID  uint64 `gorm:"not null;index"`

	// 0 表示一级评论
	ParentID uint64 `gorm:"not null;default:0;index"`

	Content string `gorm:"type:text;not null"`

	LikeCount int64 `gorm:"not null;default:0"`

	CreatedAt time.Time `gorm:"index:idx_comment_video_created,priority:2"`
	UpdatedAt time.Time
}

func (Comment) TableName() string {
	return "comment"
}
