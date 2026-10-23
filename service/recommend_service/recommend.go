package recommend_service

import (
	"context"
	"errors"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"go-douyin/service/hot_service"
	"math"
	"sort"
	"time"
)

const (
	candidateLimit = 500
	maxPageSize    = 50
)

var ErrInvalidUser = errors.New("用户ID不合法")

type likedCategory struct {
	Category string `gorm:"column:category"`
}

type rankedVideo struct {
	Video models.Video
	Score float64
}

// LoadInterests 统计用户最近点赞的视频类别。
// 使用video_likes表，不需要新增兴趣表。
func LoadInterests(
	ctx context.Context,
	userID uint64,
) (map[string]int, error) {
	var rows []likedCategory

	err := database.DB.WithContext(ctx).
		Table("video_like AS l").
		Select("v.category AS category").
		Joins(
			"JOIN video AS v ON v.id = l.video_id",
		).
		Where(
			"l.user_id = ? AND v.status = ?",
			userID, 1,
		).
		Order("l.created_at DESC").
		Order("l.id DESC").
		Limit(200).
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	interests := make(map[string]int)

	for _, row := range rows {
		if row.Category == "" ||
			row.Category == "general" {
			continue
		}

		interests[row.Category]++
	}

	return interests, nil
}

// InterestScore 将某分类的点赞次数映射到0～1。
func InterestScore(count int) float64 {
	if count <= 0 {
		return 0
	}
	if count >= 5 {
		return 1
	}
	return float64(count) / 5
}

// CalculateRecommendScore 计算推荐分数。
// 有局限性 只是在随机500个视频中找到自己是否有喜欢的分类的视频类型
// 后续可以引申 各个分类的 热点视频 推荐确实需要改正 否则随机500 不一定作者能刷到自己分类的
func CalculateRecommendScore(video models.Video,
	interests map[string]int,
	now time.Time,
) float64 {
	hot := hot_service.CalculateHotScore(
		video.LikeCount,
		video.CommentCount,
		video.CreatedAt,
		now,
	)
	interest := InterestScore(
		interests[video.Category],
	)
	return math.Log1p(hot) + 0.8*interest
}

// RankVideos 对候选视频进行推荐排序。
func RankVideos(
	videos []models.Video,
	interests map[string]int,
	now time.Time,
) []rankedVideo {
	ranked := make([]rankedVideo, 0, len(videos))
	for _, video := range videos {
		ranked = append(ranked, rankedVideo{
			Video: video,
			Score: CalculateRecommendScore(video, interests, now),
		})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Video.ID >
				ranked[j].Video.ID
		}
		return ranked[i].Score > ranked[j].Score
	})

	return ranked
}

// ListRecommendFeed 返回个性化推荐视频。
func ListRecommendFeed(
	ctx context.Context,
	userID uint64,
	limit int,
	rawCursor string,
) (dto.RecommendFeedResponse, error) {
	result := dto.RecommendFeedResponse{
		List: make([]dto.RecommendVideoItem, 0),
	}

	if userID == 0 {
		return result, ErrInvalidUser
	}

	if limit < 1 {
		limit = 10
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	// 第一页使用当前时间。
	// 后续页面沿用游标中的基准时间。
	asOf := time.Now().UTC()

	var cursor recommendCursor
	if rawCursor != "" {
		var err error
		cursor, err = decodeCursor(rawCursor, userID)
		if err != nil {
			return result, err
		}
		asOf = cursor.AsOf
	}

	// 1. 查询用户点赞兴趣
	interests, err := LoadInterests(ctx, userID)
	if err != nil {
		return result, err
	}
	// 是否进行分类热点
	result.ColdStart = len(interests) == 0

	// 2. 查询近期已经点赞的视频ID
	likedIDs, err := LoadRecentLikedIDs(ctx, userID)
	if err != nil {
		return result, err
	}

	// 3. 根据用户兴趣决定召回路径
	// 有兴趣：兴趣 + 热门 + 最新
	// 无兴趣：热门 + 最新
	videos, err := RecallCandidates(
		ctx,
		interests,
		likedIDs,
		asOf,
	)
	if err != nil {
		return result, err
	}

	// 3. 计算推荐分数并排序
	ranked := RankVideos(videos, interests, asOf)

	// 4. 根据上一页游标找到起点
	start := 0

	if rawCursor != "" {
		start = len(ranked)

		for i, item := range ranked {
			if item.Score < cursor.Score ||
				(item.Score == cursor.Score &&
					item.Video.ID < cursor.ID) {
				start = i
				break
			}
		}
	}

	end := start + limit
	if end > len(ranked) {
		end = len(ranked)
	}

	pageItems := ranked[start:end]
	result.HasMore = end < len(ranked)

	// 5. 转换成返回给前端的DTO
	for _, item := range pageItems {
		v := item.Video

		result.List = append(
			result.List,
			dto.RecommendVideoItem{
				ID:           v.ID,
				AuthorID:     v.AuthorID,
				Title:        v.Title,
				Description:  v.Description,
				Category:     v.Category,
				PlayURL:      v.PlayURL,
				CoverURL:     v.CoverURL,
				LikeCount:    v.LikeCount,
				CommentCount: v.CommentCount,
				Score:        item.Score,
			},
		)
	}

	// 6. 生成下一页游标
	if result.HasMore && len(pageItems) > 0 {
		last := pageItems[len(pageItems)-1]

		result.NextCursor, err = encodeCursor(
			recommendCursor{
				UserID: userID,
				AsOf:   asOf,
				Score:  last.Score,
				ID:     last.Video.ID,
			},
		)
		if err != nil {
			return dto.RecommendFeedResponse{}, err
		}
	}

	return result, nil
}
