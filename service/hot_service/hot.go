package hot_service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"sort"
	"strconv"
	"time"
)

const (
	hotKey          = "douyin:videos:hot"
	hotTTL          = 2 * time.Hour
	hotCandidateMax = 5000
	hotResultMax    = 50
)

type scoredVideo struct {
	Video models.Video
	Score float64
}

// 选择最近30天、最近发布的最多5000条公开视频。
// 候选集合是有界的，避免每次扫描整个视频表。
func loadCandidates(
	ctx context.Context,
) ([]models.Video, error) {
	var videos []models.Video

	err := database.DB.WithContext(ctx).
		Model(&models.Video{}).
		Where(
			"status = ? AND created_at >= ?",
			1, time.Now().Add(-30*24*time.Hour),
		).
		Order("created_at DESC").
		Order("id DESC").
		Limit(hotCandidateMax).
		Find(&videos).Error
	return videos, err
}

// 计算并排序候选视频。
func rankCandidates(
	videos []models.Video,
	now time.Time,
) []scoredVideo {
	ranked := make([]scoredVideo, 0, len(videos))
	for _, video := range videos {
		ranked = append(ranked, scoredVideo{
			Video: video,
			Score: CalculateHotScore(
				video.LikeCount,
				video.CommentCount,
				video.CreatedAt,
				now,
			),
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

// RebuildHotRank 重新生成Redis排行榜。
func RebuildHotRank(ctx context.Context) error {
	// 找出近30天视频
	videos, err := loadCandidates(ctx)
	if err != nil {
		return fmt.Errorf("加载视频候选失败: %w", err)
	}
	ranked := rankCandidates(videos, time.Now())
	token := make([]byte, 8)
	if _, err := rand.Read(token); err != nil {
		return err
	}

	tempKey := hotKey + ":tmp:" + hex.EncodeToString(token)
	// 使用临时Key，避免直接清空正式排行榜。
	if len(ranked) > 0 {
		members := make([]redis.Z, 0, len(ranked))
		for _, item := range ranked {
			members = append(members, redis.Z{
				Score:  item.Score,
				Member: strconv.FormatUint(item.Video.ID, 10),
			})
		}

		if err := database.Redis.ZAdd(
			ctx, tempKey, members...,
		).Err(); err != nil {
			return err
		}

		// 临时Key即使未能切换，也会自动清理。
		if err := database.Redis.Expire(
			ctx, tempKey, 5*time.Minute,
		).Err(); err != nil {
			return err
		}
	}

	// 一次性切换正式榜单，避免读到半成品。
	// 当没有公开视频时，删除旧榜单。
	swapScript := redis.NewScript(`
if tonumber(ARGV[2]) == 0 then
    redis.call("DEL", KEYS[1])
    return 1
end
if redis.call("EXISTS", KEYS[2]) == 0 then
    return redis.error_reply("temporary rank missing")
end
redis.call("RENAME", KEYS[2], KEYS[1])
redis.call("PEXPIRE", KEYS[1], ARGV[1])
return 1
`)

	err = swapScript.Run(
		ctx,
		database.Redis,
		[]string{hotKey, tempKey},
		hotTTL.Milliseconds(),
		len(ranked),
	).Err()

	if err != nil {
		return fmt.Errorf("切换热榜失败: %w", err)
	}

	return nil
}

// 从Redis读取排行榜中的视频ID及对应分数。
func loadRankFromRedis(
	ctx context.Context,
) ([]redis.Z, error) {
	// 缓存不存在时由调用方降级到MySQL。
	exists, err := database.Redis.Exists(
		ctx, hotKey,
	).Result()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, redis.Nil
	}

	return database.Redis.ZRevRangeWithScores(
		ctx, hotKey, 0, hotCandidateMax-1,
	).Result()
}

// ListHotVideos 对外提供排行榜查询。
func ListHotVideos(
	ctx context.Context,
	limit int,
) (dto.HotVideoResponse, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > hotResultMax {
		limit = hotResultMax
	}
	// 从redis中找出视频排序id
	ranked, err := loadRankFromRedis(ctx)

	if err != nil {
		// Redis异常或榜单不存在：走MySQL降级。
		return listHotFromMySQL(ctx, limit)
	}

	ids := make([]uint64, 0, len(ranked))
	scoreByID := make(map[uint64]float64, len(ranked))
	// 从 redis中获取热点视频id
	for _, z := range ranked {
		rawID, ok := z.Member.(string)
		if !ok {
			continue
		}
		id, err := strconv.ParseUint(rawID, 10, 64)
		if err != nil {
			continue
		}

		ids = append(ids, id)
		scoreByID[id] = z.Score
	}

	result := dto.HotVideoResponse{
		List:   make([]dto.HotVideoItem, 0, limit),
		Source: "redis",
	}

	if len(ids) == 0 {
		return result, nil
	}

	// 用主键批量获取视频详情。
	// 必须再次检查status，防止下架视频被展示。
	var videos []models.Video
	err = database.DB.WithContext(ctx).
		Where("id IN ? AND status = ?", ids, 1).
		Find(&videos).Error

	if err != nil {
		return dto.HotVideoResponse{}, err
	}

	videoByID := make(map[uint64]models.Video, len(videos))
	for _, v := range videos {
		videoByID[v.ID] = v
	}

	// 按Redis排行顺序重新组装
	for _, id := range ids {
		v, ok := videoByID[id]
		if !ok {
			continue
		}

		result.List = append(result.List, toHotItem(
			v, scoreByID[id],
		))

		if len(result.List) >= limit {
			break
		}
	}

	return result, nil
}

// 从mysql进行查找 需要在进行筛选
func listHotFromMySQL(
	ctx context.Context,
	limit int,
) (dto.HotVideoResponse, error) {
	// 从近30天中找出视频
	videos, err := loadCandidates(ctx)
	if err != nil {
		return dto.HotVideoResponse{}, err
	}
	// 把30天所选视频进行排序
	ranked := rankCandidates(videos, time.Now())

	result := dto.HotVideoResponse{
		List:   make([]dto.HotVideoItem, 0, limit),
		Source: "mysql",
	}

	for i, item := range ranked {
		if i >= limit {
			break
		}
		result.List = append(
			result.List,
			toHotItem(item.Video, item.Score),
		)
	}

	return result, nil
}

func toHotItem(
	video models.Video,
	score float64,
) dto.HotVideoItem {
	return dto.HotVideoItem{
		ID:           video.ID,
		AuthorID:     video.AuthorID,
		Title:        video.Title,
		Description:  video.Description,
		PlayURL:      video.PlayURL,
		CoverURL:     video.CoverURL,
		LikeCount:    video.LikeCount,
		CommentCount: video.CommentCount,
		HotScore:     score,
	}
}
