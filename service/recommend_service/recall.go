package recommend_service

import (
	"context"
	"go-douyin/database"
	"go-douyin/models"
	"gorm.io/gorm"
	"sort"
	"time"
)

// 有兴趣用户的三路候选上限
const (
	interestRecallLimit = 300
	hotRecallLimit      = 125
	latestRecallLimit   = 75
)

// 新用户没有兴趣时，重新分配额度
const (
	coldHotLimit    = 300
	coldLatestLimit = 200
)

// TopCategories 取用户最喜欢的前3个分类
func TopCategories(
	interests map[string]int,
) []string {
	type categoryCount struct {
		Name  string
		Count int
	}

	var categories []categoryCount

	for name, count := range interests {
		if name == "" || count <= 0 {
			continue
		}

		categories = append(categories, categoryCount{
			Name:  name,
			Count: count,
		})
	}

	sort.Slice(categories, func(i, j int) bool {
		if categories[i].Count == categories[j].Count {
			return categories[i].Name < categories[j].Name
		}
		return categories[i].Count > categories[j].Count
	})

	if len(categories) > 3 {
		categories = categories[:3]
	}

	result := make([]string, 0, len(categories))
	for _, item := range categories {
		result = append(result, item.Name)
	}

	return result

}

// LoadRecentLikedIDs 查询最近200条点赞关系。
// 用于过滤已经点赞过的视频。
func LoadRecentLikedIDs(
	ctx context.Context,
	userID uint64,
) (map[uint64]struct{}, error) {
	var ids []uint64

	err := database.DB.WithContext(ctx).
		Table("video_like").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Order("id DESC").
		Limit(200).
		Pluck("video_id", &ids).Error

	if err != nil {
		return nil, err
	}

	liked := make(map[uint64]struct{}, len(ids))

	for _, id := range ids {
		liked[id] = struct{}{}
	}

	return liked, nil
}

// MergeCandidates 合并多个来源的视频。
// 根据视频ID去重，并排除已点赞的视频。
func MergeCandidates(
	groups [][]models.Video,
	excluded map[uint64]struct{},
) []models.Video {
	result := make([]models.Video, 0)
	seen := make(map[uint64]struct{})
	for _, group := range groups {
		for _, video := range group {
			if video.ID == 0 {
				continue
			}
			if _, liked := excluded[video.ID]; liked {
				continue
			}
			if _, exists := seen[video.ID]; exists {
				continue
			}

			seen[video.ID] = struct{}{}
			result = append(result, video)
		}
	}

	return result
}

// RecallCandidates 执行多路召回。
// asOf 是推荐请求的统一时间基准。
func RecallCandidates(
	ctx context.Context,
	interests map[string]int,
	excluded map[uint64]struct{},
	asOf time.Time,
) ([]models.Video, error) {
	db := database.DB.WithContext(ctx)

	// 所有路径都只查询已经发布的公开视频。
	baseQuery := func(days int) *gorm.DB {
		return db.Model(&models.Video{}).
			Where("status = ?", 1).
			Where("created_at <= ?", asOf).
			Where(
				"created_at >= ?",
				asOf.AddDate(0, 0, -days),
			)
	}
	var groups [][]models.Video

	categories := TopCategories(interests)
	if len(categories) > 0 {
		// 路径1：按用户最喜欢的类别召回。
		// 允许搜索最近180天，而不只是最新500条。
		var interestVideos []models.Video

		err := baseQuery(180).
			Where("category IN ?", categories).
			Order(
				"(like_count + 3 * comment_count) DESC",
			).
			Order("created_at DESC").
			Order("id DESC").
			Limit(interestRecallLimit).
			Find(&interestVideos).Error

		if err != nil {
			return nil, err
		}
		groups = append(groups, interestVideos)
	}
	// 路径2：热门召回
	hotLimit := hotRecallLimit
	if len(categories) == 0 {
		hotLimit = coldHotLimit
	}
	var hotVideos []models.Video

	err := baseQuery(30).
		Order(
			"(like_count + 3 * comment_count) DESC",
		).
		Order("created_at DESC").
		Order("id DESC").
		Limit(hotLimit).
		Find(&hotVideos).Error
	if err != nil {
		return nil, err
	}

	groups = append(groups, hotVideos)
	// 路径3：最新视频召回
	latestLimit := latestRecallLimit
	if len(categories) == 0 {
		latestLimit = coldLatestLimit
	}

	var latestVideos []models.Video

	err = baseQuery(30).
		Order("created_at DESC").
		Order("id DESC").
		Limit(latestLimit).
		Find(&latestVideos).Error
	if err != nil {
		return nil, err
	}

	groups = append(groups, latestVideos)

	// 召回顺序不等于最终推荐顺序。
	// 这里只负责合并、去重和过滤。
	return MergeCandidates(groups, excluded), nil
}
