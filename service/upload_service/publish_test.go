package upload_service

import (
	"errors"
	"testing"
)

func TestValidatePublishInput(t *testing.T) {
	tests := []struct {
		name    string
		input   PublishUploadedInput
		wantErr bool
	}{
		{
			name: "正常发布",
			input: PublishUploadedInput{
				Title:     "Go后端项目",
				Category:  "tech",
				ObjectKey: "videos/1/test.mp4",
			},
		},
		{
			name: "标题为空",
			input: PublishUploadedInput{
				Title:     "  ",
				Category:  "tech",
				ObjectKey: "videos/1/test.mp4",
			},
			wantErr: true,
		},
		{
			name: "分类不合法",
			input: PublishUploadedInput{
				Title:     "测试视频",
				Category:  "unknown",
				ObjectKey: "videos/1/test.mp4",
			},
			wantErr: true,
		},
		{
			name: "对象Key为空",
			input: PublishUploadedInput{
				Title:    "测试视频",
				Category: "tech",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePublishInput(tt.input)

			if tt.wantErr {
				if !errors.Is(err, ErrBadPublish) {
					t.Fatalf(
						"期望参数校验失败，实际%v",
						err,
					)
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
