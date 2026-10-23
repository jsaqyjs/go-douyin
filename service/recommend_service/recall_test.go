package recommend_service

import (
	"reflect"
	"testing"

	"go-douyin/models"
)

func TestTopCategories(t *testing.T) {
	interests := map[string]int{
		"tech":   10,
		"game":   6,
		"food":   3,
		"travel": 1,
	}

	got := TopCategories(interests)
	want := []string{"tech", "game", "food"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"期望%v，实际%v",
			want, got,
		)
	}
}

func TestTopCategoriesColdStart(t *testing.T) {
	interests := map[string]int{}

	got := TopCategories(interests)

	if len(got) != 0 {
		t.Fatalf("无兴趣时应返回空分类，实际%v", got)
	}
}

func TestMergeCandidates(t *testing.T) {
	// 三路召回结果中故意包含重复视频
	interestVideos := []models.Video{
		{ID: 1, Title: "Go教程"},
		{ID: 2, Title: "MySQL教程"},
		{ID: 3, Title: "Redis教程"},
	}

	hotVideos := []models.Video{
		{ID: 2, Title: "MySQL教程"},
		{ID: 4, Title: "热门游戏"},
	}

	latestVideos := []models.Video{
		{ID: 3, Title: "Redis教程"},
		{ID: 5, Title: "最新视频"},
	}

	// 用户已经点赞视频2
	likedIDs := map[uint64]struct{}{
		2: {},
	}

	result := MergeCandidates(
		[][]models.Video{
			interestVideos,
			hotVideos,
			latestVideos,
		},
		likedIDs,
	)

	// 视频2被排除，视频3不会重复出现
	want := []uint64{1, 3, 4, 5}

	if len(result) != len(want) {
		t.Fatalf(
			"期望%d条视频，实际%d条",
			len(want), len(result),
		)
	}

	for i, id := range want {
		if result[i].ID != id {
			t.Fatalf(
				"第%d条期望ID=%d，实际ID=%d",
				i, id, result[i].ID,
			)
		}
	}
}

func TestMergeCandidatesEmpty(t *testing.T) {
	result := MergeCandidates(
		[][]models.Video{},
		map[uint64]struct{}{},
	)

	if len(result) != 0 {
		t.Fatal("没有候选视频时应返回空结果")
	}
}
