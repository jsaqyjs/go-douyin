package event_service

import (
	"errors"
	"testing"
)

func TestDecodeLikeEvent(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "正常事件",
			body: `{
                "event_id":"like-10001",
                "video_id":15,
                "user_id":3
            }`,
		},
		{
			name:    "无效JSON",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name: "缺少EventID",
			body: `{
                "video_id":15,
                "user_id":3
            }`,
			wantErr: true,
		},
		{
			name: "VideoID为零",
			body: `{
                "event_id":"like-10001",
                "video_id":0,
                "user_id":3
            }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeLikeEvent(
				[]byte(tt.body),
			)

			if tt.wantErr {
				if !errors.Is(err, ErrInvalidEvent) {
					t.Fatalf(
						"期望ErrInvalidEvent，实际%v",
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
