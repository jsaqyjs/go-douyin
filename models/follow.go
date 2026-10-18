package models

import "time"

type Follow struct {
	ID          uint64 `gorm:"primary_key;AUTO_INCREMENT"`
	FollowerID  uint64 `gorm:"not null;uniqueIndex:uk_follower_following,priority:1;index:idx_following_created,priority:1"`
	FollowingID uint64 `gorm:"not null;uniqueIndex:uk_following_following,priority:2"`
	CreatedAt   time.Time
}

func (Follow) TableName() string {
	return "follow"
}
