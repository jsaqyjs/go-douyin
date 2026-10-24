package upload_service

import (
	"context"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"go-douyin/database"
	"go-douyin/models"
	"go-douyin/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

var (
	ErrAssetNotFound    = errors.New("上传文件不存在")
	ErrAssetForbidden   = errors.New("不能使用其他用户的文件")
	ErrAssetUnavailable = errors.New("文件已经发布或正在删除")
	ErrBadPublish       = errors.New("发布参数不合法")
)

type PublishUploadedInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	ObjectKey   string `json:"object_key"`
}

// ValidatePublishInput 负责请求参数的业务校验。
func ValidatePublishInput(in PublishUploadedInput) error {
	title := strings.TrimSpace(in.Title)
	if len([]rune(title)) < 1 ||
		len([]rune(title)) > 100 {
		return ErrBadPublish
	}
	if len([]rune(in.Description)) > 1000 {
		return ErrBadPublish
	}
	if len(in.ObjectKey) == 0 ||
		len(in.ObjectKey) > 255 {
		return ErrBadPublish
	}

	switch in.Category {
	case "general", "tech", "game",
		"food", "travel", "music":
		return nil
	default:
		return ErrBadPublish
	}
}
func PublishUploadedVideo(
	ctx context.Context,
	userID uint64,
	in PublishUploadedInput,
) (uint64, error) {
	if userID == 0 {
		return 0, ErrAssetForbidden
	}

	if err := ValidatePublishInput(in); err != nil {
		return 0, err
	}
	if storage.MinIO == nil {
		return 0, errors.New("MinIO未初始化")
	}

	var videoID uint64
	err := database.DB.WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {

			// 1. 行级锁：同一个文件只能被一个事务处理。
			var asset models.UploadAsset
			err := tx.Clauses(
				clause.Locking{Strength: "UPDATE"},
			).Where("object_key = ?", in.ObjectKey).First(&asset).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAssetNotFound
			}
			if err != nil {
				return err
			}
			// 2. 必须使用JWT中解析出的用户ID。
			if asset.UserID != userID {
				return ErrAssetForbidden
			}
			// 3. 不能再次发布已经使用的文件。
			if asset.Status != models.AssetUploaded {
				return ErrAssetUnavailable
			}

			// 4. 检查私有对象存储里的文件。
			statCtx, cancel := context.WithTimeout(
				ctx,
				5*time.Second,
			)
			defer cancel()
			info, err := storage.MinIO.Client.StatObject(
				statCtx,
				storage.MinIO.Bucket,
				asset.ObjectKey,
				minio.StatObjectOptions{},
			)
			if err != nil {
				return fmt.Errorf(
					"检查对象存储失败: %w", err,
				)
			}

			if info.Size <= 0 || info.Size != asset.Size {
				return ErrAssetUnavailable
			}

			// 5. 创建正式的视频记录。
			video := models.Video{
				AuthorID:    userID,
				Title:       strings.TrimSpace(in.Title),
				Description: in.Description,
				Category:    in.Category,
				PlayURL:     "minio://" + asset.ObjectKey,
				Status:      1,
			}

			if err := tx.Create(&video).Error; err != nil {
				return err
			}
			// 6. 将上传记录标记为已发布。
			result := tx.Model(&models.UploadAsset{}).
				Where(
					"id = ? AND status = ?",
					asset.ID,
					models.AssetUploaded,
				).
				Updates(map[string]interface{}{
					"status":   models.AssetPublished,
					"video_id": video.ID,
				})

			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected != 1 {
				return ErrAssetUnavailable
			}

			videoID = video.ID
			return nil
		})
	if err != nil {
		return 0, err
	}

	return videoID, nil
}
