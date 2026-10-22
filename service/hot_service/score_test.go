package hot_service

import (
	"math"
	"testing"
	"time"

	"go-douyin/models"
)

func TestCalculateHotScore(t *testing.T) {
	now := time.Date(
		2026, 10, 22,
		12, 0, 0, 0,
		time.UTC,
	)

	recent := CalculateHotScore(
		100, 10,
		now.Add(-1*time.Hour),
		now,
	)

	old := CalculateHotScore(
		100, 10,
		now.Add(-10*24*time.Hour),
		now,
	)

	if recent <= old {
		t.Fatal("较新的同互动量视频应该得分更高")
	}

	highLikes := CalculateHotScore(
		200, 10,
		now.Add(-1*time.Hour),
		now,
	)

	if highLikes <= recent {
		t.Fatal("相同发布时间下点赞更多应得分更高")
	}

	zero := CalculateHotScore(
		0, 0,
		now,
		now,
	)

	if zero != 0 {
		t.Fatalf("零互动应得0分，实际%f", zero)
	}

	if math.IsNaN(recent) || math.IsInf(recent, 0) {
		t.Fatal("热度分数不能是NaN或Inf")
	}
}

func TestRankCandidates(t *testing.T) {
	now := time.Date(
		2026, 10, 22,
		12, 0, 0, 0,
		time.UTC,
	)

	videos := []models.Video{
		{
			ID:        1,
			LikeCount: 10,
			CreatedAt: now,
		},
		{
			ID:        2,
			LikeCount: 200,
			CreatedAt: now,
		},
		{
			ID:        3,
			LikeCount: 100,
			CreatedAt: now,
		},
	}

	ranked := rankCandidates(videos, now)

	if len(ranked) != 3 {
		t.Fatalf("期望3条记录，实际%d", len(ranked))
	}

	expected := []uint64{2, 3, 1}

	for i, id := range expected {
		if ranked[i].Video.ID != id {
			t.Fatalf(
				"第%d位期望视频%d，实际%d",
				i+1, id, ranked[i].Video.ID,
			)
		}
	}
}
