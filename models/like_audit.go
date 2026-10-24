package models

import "time"

type LikeAudit struct {
	ID        uint64 `gorm:"primaryKey"`
	EventID   string `gorm:"type:varchar(64);not null;uniqueIndex"`
	VideoID   uint64 `gorm:"not null;index"`
	UserID    uint64 `gorm:"not null;index"`
	CreatedAt time.Time
}

func (LikeAudit) TableName() string {
	return "like_audit"

}
