package recommend_service

import (
	"errors"
	"testing"
	"time"

	"go-douyin/models"
)

func TestInterestScore(t *testing.T) {
	tests := []struct {
		count int
		want  float64
	}{
		{0, 0},
		{1, 0.2},
		{2, 0.4},
		{5, 1},
		{10, 1},
		{-1, 0},
	}

	for _, tt := range tests {
		got := InterestScore(tt.count)

		if got != tt.want {
			t.Errorf(
				"count=%d，期望%v，实际%v",
				tt.count, tt.want, got,
			)
		}
	}
}

func TestPersonalizedRanking(t *testing.T) {
	now := time.Date(
		2026, 10, 22,
		12, 0, 0, 0,
		time.UTC,
	)

	// 两条视频的热度完全相同
	videos := []models.Video{
		{
			ID:        1,
			Category:  "tech",
			LikeCount: 100,
			CreatedAt: now,
		},
		{
			ID:        2,
			Category:  "game",
			LikeCount: 100,
			CreatedAt: now,
		},
	}

	// 用户A偏好技术
	interestsA := map[string]int{
		"tech": 5,
	}

	rankedA := RankVideos(
		videos, interestsA, now,
	)

	if rankedA[0].Video.ID != 1 {
		t.Fatal("技术用户应优先看到技术视频")
	}

	// 用户B偏好游戏
	interestsB := map[string]int{
		"game": 5,
	}

	rankedB := RankVideos(
		videos, interestsB, now,
	)

	if rankedB[0].Video.ID != 2 {
		t.Fatal("游戏用户应优先看到游戏视频")
	}

	// 新用户没有兴趣记录，按基础热度排序。
	// 由于热度相同，按ID倒序。
	rankedCold := RankVideos(
		videos,
		map[string]int{},
		now,
	)

	if rankedCold[0].Video.ID != 2 {
		t.Fatal("冷启动情况下应按ID规则处理同分视频")
	}
}

func TestRecommendCursor(t *testing.T) {
	c := recommendCursor{
		UserID: 1,
		AsOf:   time.Now().UTC(),
		Score:  3.25,
		ID:     100,
	}

	encoded, err := encodeCursor(c)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := decodeCursor(encoded, 1)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.ID != 100 ||
		decoded.Score != 3.25 ||
		decoded.UserID != 1 {
		t.Fatal("游标编码和解码结果不一致")
	}

	// 用户1的游标不能直接用于用户2
	_, err = decodeCursor(encoded, 2)
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatal("不同用户不能混用推荐游标")
	}
}
