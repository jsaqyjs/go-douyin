package upload_service

import (
	"errors"
	"mime/multipart"
	"testing"
)

func TestIsMP4Header(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "正常MP4文件头",
			data: []byte{
				0, 0, 0, 24,
				'f', 't', 'y', 'p',
				'i', 's', 'o', 'm',
			},
			want: true,
		},
		{
			name: "错误文件头",
			data: []byte("not-a-mp4-file"),
			want: false,
		},
		{
			name: "数据太短",
			data: []byte{0, 1, 2},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMP4Header(tt.data)
			if got != tt.want {
				t.Fatalf(
					"期望%v，实际%v",
					tt.want, got,
				)
			}
		})
	}
}

func TestInvalidUpload(t *testing.T) {
	// 用户ID不能为0
	_, err := UploadVideo(
		t.Context(), 0,
		&multipart.FileHeader{
			Filename: "demo.mp4",
			Size:     1024,
		},
	)
	if !errors.Is(err, ErrInvalidVideo) {
		t.Fatalf("期望ErrInvalidVideo，实际%v", err)
	}

	// 超过最大允许大小
	_, err = UploadVideo(
		t.Context(), 1,
		&multipart.FileHeader{
			Filename: "demo.mp4",
			Size:     MaxVideoSize + 1,
		},
	)
	if !errors.Is(err, ErrVideoTooLarge) {
		t.Fatalf("期望ErrVideoTooLarge，实际%v", err)
	}

	// 不支持.txt
	_, err = UploadVideo(
		t.Context(), 1,
		&multipart.FileHeader{
			Filename: "demo.txt",
			Size:     1024,
		},
	)
	if !errors.Is(err, ErrInvalidVideo) {
		t.Fatalf("期望ErrInvalidVideo，实际%v", err)
	}
}
