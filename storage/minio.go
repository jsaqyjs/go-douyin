package storage

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"net/url"
	"time"
)

type MinIOStorage struct {
	Client *minio.Client
	Bucket string
}

var MinIO *MinIOStorage

func InitMinIO() error {
	endpoint := "127.0.0.1:9000"
	accessKey := "minio"
	secretKey := "admin111"
	bucket := "test"

	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			accessKey,
			secretKey,
			"",
		),
		Secure: false, // 本地HTTP实验环境
	})
	if err != nil {
		return fmt.Errorf("创建MinIO客户端: %w", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("检查Bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("Bucket %s 不存在", bucket)
	}

	MinIO = &MinIOStorage{
		Client: client,
		Bucket: bucket,
	}
	return nil
}

// PresignReadURL 为私有对象生成临时读取地址。
func (s *MinIOStorage) PresignReadURL(
	ctx context.Context,
	objectKey string,
) (string, error) {
	u, err := s.Client.PresignedGetObject(
		ctx,
		s.Bucket,
		objectKey,
		15*time.Minute,
		url.Values{},
	)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
