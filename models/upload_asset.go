package models

import "time"

// uploaded	文件已经上传，但还没有发布
// published	文件已经绑定到一个视频
// deleting	文件已经进入清理流程，不能再发布
const (
	AssetUploaded  = "upload"
	AssetPublished = "published"
	AssetDeleting  = "deleting"
)

type UploadAsset struct {
	ID uint64 `gorm:"primaryKey"`

	UserID uint64 `gorm:"not null;index:idx_asset_owner"`

	ObjectKey string `gorm:"type:varchar(255);not null;uniqueIndex"`

	Size int64 `gorm:"not null"`

	Status string `gorm:"type:varchar(20);not null;index:idx_asset_status_created,priority:1"`

	VideoID uint64 `gorm:"not null;default:0"`

	CreatedAt time.Time `gorm:"index:idx_asset_status_created,priority:2"`
	UpdatedAt time.Time
}

func (UploadAsset) TableName() string {
	return "upload_asset"
}
