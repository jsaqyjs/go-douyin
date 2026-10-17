package models

import "time"

type Follow struct {
	ID          uint64 `gorm:"primary_key;AUTO_INCREMENT"`
	FollowID    uint64 `gorm:"not null;uniqueIndex:uk_follower_following,priority:1"`
	FollowingID uint64 `gorm:"not null;uniqueIndex:uk_following_following,priority:2"`
	CreateAt    time.Time
}

func (Follow) TableName() string {
	return "follow"
}
