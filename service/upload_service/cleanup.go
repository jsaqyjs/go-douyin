package upload_service

import (
	"context"
	"errors"
	"github.com/minio/minio-go/v7"
	"go-douyin/database"
	"go-douyin/models"
	"go-douyin/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log/slog"
	"time"
)

const uploadRetention = 24 * time.Hour

// claimExpiredAsset 在事务中认领待清理文件。
// 若已发布或尚未过期，不会认领。
func claimExpiredAsset(
	ctx context.Context,
	id uint64,
	cutoff time.Time,
) (models.UploadAsset, bool, error) {
	var asset models.UploadAsset
	claimed := false
	err := database.DB.WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			err := tx.Clauses(
				clause.Locking{Strength: "UPDATE"},
			).
				First(&asset, id).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			// 允许重试之前已认领、但删除失败的任务。
			if asset.Status == models.AssetDeleting {
				claimed = true
				return nil
			}

			if asset.Status != models.AssetUploaded ||
				asset.CreatedAt.After(cutoff) {
				return nil
			}

			result := tx.Model(&asset).
				Update("status", models.AssetDeleting)

			if result.Error != nil {
				return result.Error
			}

			claimed = result.RowsAffected == 1
			return nil
		})
	return asset, claimed, err
}

// CleanupExpiredUploads 清理过期的未发布对象。
// 建议由单实例Worker定期执行。
func CleanupExpiredUploads(ctx context.Context) error {
	cutoff := time.Now().Add(-uploadRetention)
	// 单次最多处理100条，避免一次清理过多。
	var candidates []models.UploadAsset
	err := database.DB.WithContext(ctx).
		Where(
			"(status = ? AND created_at < ?) OR status = ?",
			models.AssetUploaded,
			cutoff,
			models.AssetDeleting,
		).
		Order("created_at ASC").
		Limit(100).
		Find(&candidates).Error
	if err != nil {
		return err
	}

	s := storage.MinIO
	if s == nil {
		return errors.New("MinIO未初始化")
	}
	for _, item := range candidates {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		asset, claimed, err := claimExpiredAsset(
			ctx, item.ID, cutoff,
		)
		if err != nil {
			slog.Error("认领待清理对象失败",
				"id", item.ID,
				"error", err,
			)
			continue
		}
		if !claimed {
			continue
		}
		deleteCtx, cancel := context.WithTimeout(
			ctx, 10*time.Second,
		)
		err = s.Client.RemoveObject(
			deleteCtx,
			s.Bucket,
			asset.ObjectKey,
			minio.RemoveObjectOptions{},
		)
		cancel()

		if err != nil {
			// 保留 deleting，后续运行时重试。
			slog.Error("删除MinIO对象失败",
				"id", asset.ID,
				"error", err,
			)
			continue
		}
		// 对象删除成功后，再删除上传记录。
		// 如果数据库失败，后续可再次执行清理。
		err = database.DB.WithContext(ctx).
			Where(
				"id = ? AND status = ?",
				asset.ID,
				models.AssetDeleting,
			).
			Delete(&models.UploadAsset{}).Error

		if err != nil {
			slog.Error("删除上传登记失败",
				"id", asset.ID,
				"error", err,
			)
		}
	}

	return nil

}

func RunCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := CleanupExpiredUploads(ctx); err != nil {
				slog.Error(
					"定期清理未发布视频失败",
					"error", err,
				)
			}
		}
	}
}
