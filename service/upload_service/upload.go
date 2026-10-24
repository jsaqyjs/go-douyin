package upload_service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"go-douyin/database"
	"go-douyin/models"
	"go-douyin/storage"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const MaxVideoSize int64 = 100 << 20

var (
	ErrInvalidVideo  = errors.New("视频文件不合法")
	ErrVideoTooLarge = errors.New("视频超过100MiB")
)

type UploadResult struct {
	ObjectKey string `json:"object_key"`
	PlayURL   string `json:"play_url"`
	Size      int64  `json:"size"`
}

// IsMP4Header 检查MP4文件头的ftyp标识。
// 注意：这只是格式初筛，不是完整媒体解码验证。
func IsMP4Header(header []byte) bool {
	// 为什么要测长度呢 应为 ISO基础媒体文件格式决定 它由12个字节组成 然后呢 4-7是
	// .mp4 格式身份证
	return len(header) >= 12 &&
		bytes.Equal(header[4:8], []byte("ftyp"))
}
func UploadVideo(
	ctx context.Context,
	userID uint64,
	fileHeader *multipart.FileHeader,
) (UploadResult, error) {
	var result UploadResult
	if userID == 0 || fileHeader == nil {
		return result, ErrInvalidVideo
	}

	if fileHeader.Size <= 0 {
		return result, ErrInvalidVideo
	}
	if fileHeader.Size > MaxVideoSize {
		return result, ErrVideoTooLarge
	}
	if !strings.EqualFold(
		filepath.Ext(fileHeader.Filename), ".mp4",
	) {
		return result, ErrInvalidVideo
	}

	file, err := fileHeader.Open()
	if err != nil {
		return result, err
	}
	defer file.Close()
	header := make([]byte, 12)
	if _, err = io.ReadFull(file, header); err != nil {
		return result, ErrInvalidVideo
	}
	if !IsMP4Header(header) {
		return result, ErrInvalidVideo
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return result, err
	}
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return result, err
	}
	objectKey := fmt.Sprintf(
		"videos/%s/%s.mp4",
		strconv.FormatUint(userID, 10),
		hex.EncodeToString(randomBytes),
	)
	log.Printf("upload video %s to %s", objectKey, fileHeader.Filename)
	// client
	s := storage.MinIO
	if s == nil {
		return result, errors.New("MinIO未初始化")
	}

	// 使用固定大小的对象上传，而不是先读取整个视频。
	_, err = s.Client.PutObject(
		ctx,
		s.Bucket,
		objectKey,
		file,
		fileHeader.Size,
		minio.PutObjectOptions{
			ContentType: "video/mp4",
		},
	)
	if err != nil {
		return result, fmt.Errorf("上传MinIO失败: %w", err)
	}
	asset := models.UploadAsset{
		UserID:    userID,
		ObjectKey: objectKey,
		Size:      fileHeader.Size,
		Status:    models.AssetUploaded,
	}
	err = database.DB.WithContext(ctx).
		Create(&asset).Error
	if err != nil {
		// MinIO上传成功，但数据库登记失败。
		// 用独立超时Context尽力删除文件。
		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		_ = s.Client.RemoveObject(
			cleanupCtx,
			s.Bucket,
			objectKey,
			minio.RemoveObjectOptions{},
		)

		return result, fmt.Errorf(
			"登记上传文件失败: %w", err,
		)
	}
	playURL, err := s.PresignReadURL(ctx, objectKey)
	if err != nil {
		// 上传成功但临时URL生成失败：
		// 尝试回滚已上传的对象。
		_ = s.Client.RemoveObject(
			ctx, s.Bucket, objectKey,
			minio.RemoveObjectOptions{},
		)
		return result, fmt.Errorf("生成播放地址失败: %w", err)
	}
	log.Print("离成功差一步")
	return UploadResult{
		ObjectKey: objectKey,
		PlayURL:   playURL,
		Size:      fileHeader.Size,
	}, nil
}
