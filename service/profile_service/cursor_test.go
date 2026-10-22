package profile_service

import (
	"errors"
	"go-douyin/utils"
	"testing"
	"time"
)

var ErrInvalidCursor = errors.New("无效的游标")

func TestVideoCursorEncodeDecode(t *testing.T) {
	createdAt := time.Date(
		2026, 10, 22,
		10, 20, 0, 123000000,
		time.UTC,
	)

	encoded, err := utils.EncodeFeedCursor(
		createdAt, 107,
	)
	if err != nil {
		t.Fatal(err)
	}

	if encoded == "" {
		t.Fatal("编码结果不能为空")
	}

	decoded, err := utils.DecodeFeedCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.ID != 107 {
		t.Fatalf(
			"ID错误，期望107，实际%d",
			decoded.ID,
		)
	}

	if !decoded.Time.Equal(createdAt) {
		t.Fatalf("时间解码后不一致")
	}
}

func TestInvalidVideoCursor(t *testing.T) {
	cases := []string{
		"",
		"invalid!!!",
		"e30", // Base64编码后的 {}
	}

	for _, raw := range cases {
		_, err := utils.DecodeFeedCursor(raw)

		if !errors.Is(err, ErrInvalidCursor) {
			t.Errorf(
				"游标%q应返回ErrInvalidCursor",
				raw,
			)
		}
	}
}

// 测试排序时间相同时，下一页必须按ID继续。
func TestVideoCursorSameTimestamp(t *testing.T) {
	createdAt := time.Date(
		2026, 10, 22,
		10, 20, 0, 0,
		time.UTC,
	)

	// 第一页最后一个视频ID为107
	encoded, err := utils.EncodeFeedCursor(createdAt, 107)
	if err != nil {
		t.Fatal(err)
	}

	cursor, err := utils.DecodeFeedCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}

	// 用与SQL相同的条件验证比较逻辑
	candidates := []struct {
		ID        uint64
		CreatedAt time.Time
	}{
		{108, createdAt},
		{107, createdAt},
		{106, createdAt},
		{105, createdAt},
	}

	var nextIDs []uint64

	for _, video := range candidates {
		isNext := video.CreatedAt.Before(cursor.Time) ||
			(video.CreatedAt.Equal(cursor.Time) &&
				video.ID < cursor.ID)

		if isNext {
			nextIDs = append(nextIDs, video.ID)
		}
	}

	if len(nextIDs) != 2 ||
		nextIDs[0] != 106 ||
		nextIDs[1] != 105 {

		t.Fatalf(
			"下一页结果错误: %v",
			nextIDs,
		)
	}
}
