package models

import "time"

const (
	VideoStatusReview    uint8 = 0
	VideoStatusPublished uint8 = 1
	VideoStatusOffline   uint8 = 2
)

// Video 用户 ID：1
// 视频 ID：100
// 标题：今天去南京旅游
// 视频地址：https://example.com/100.mp4
// 点赞数：356
// 评论数：28
type Video struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	AuthorID uint64 `gorm:"not null;index:idx_video_author_created,priority:1"`

	Title       string `gorm:"size:200;not null"`
	Description string `gorm:"type:text"`
	PlayURL     string `gorm:"size:1024;not null"`
	CoverURL    string `gorm:"size:1024"`

	LikeCount    int64 `gorm:"not null;default:0"`
	CommentCount int64 `gorm:"not null;default:0"`

	Status uint8 `gorm:"not null;default:1"`

	CreatedAt time.Time `gorm:"index:idx_video_author_created,priority:2,sort:desc"`
	UpdatedAt time.Time
}

func (Video) TableName() string {
	return "video"
}
