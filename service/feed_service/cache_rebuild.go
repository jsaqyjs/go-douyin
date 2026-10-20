package feed_service

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go-douyin/database"
	"go-douyin/models"
	"strconv"
	"time"
)

const feedReadyTTL = 60 * time.Second

func feedReadyKey(userID uint64) string {
	return feedKey(userID) + ":ready"
}

// RebuildFollowingFeed 从MySQL重建Redis收件箱
func RebuildFollowingFeed(
	ctx context.Context,
	userID uint64,
) error {
	if database.Redis == nil {
		return fmt.Errorf("Redis未初始化")
	}

	// 1. 查询最近1000条关注视频
	var videos []models.Video

	err := database.DB.WithContext(ctx).
		Model(&models.Video{}).
		Joins(
			"JOIN follow AS f ON "+
				"f.following_id = video.author_id",
		).
		Where("f.follower_id = ?", userID).
		Where(
			"video.status = ?",
			models.VideoStatusPublished,
		).
		Select("video.*").
		Order("video.created_at DESC").
		Order("video.id DESC").
		Limit(feedMaxItems).
		Find(&videos).Error
	if err != nil {
		return fmt.Errorf("查询关注视频失败: %w", err)
	}

	// 2. 构造ZSet成员
	members := make([]redis.Z, 0, len(videos))
	for _, video := range videos {
		members = append(members, redis.Z{
			Score:  float64(video.CreatedAt.UnixMilli()),
			Member: strconv.FormatUint(video.ID, 10),
		})
	}
	// 3. 原子提交Redis缓存重建命令
	key := feedKey(userID)
	readyKey := feedReadyKey(userID)
	_, err = database.Redis.TxPipelined(
		ctx,
		func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, readyKey)
			pipe.Del(ctx, key)

			if len(members) > 0 {
				pipe.ZAdd(ctx, key, members...)
				pipe.Expire(ctx, key, feedTTL)
			}

			pipe.Set(ctx, readyKey, "1", feedReadyTTL)

			return nil
		},
	)

	if err != nil {
		return fmt.Errorf("重建Redis Feed失败: %w", err)
	}

	return nil
}

// InvalidateFollowingFeed 使用户收件箱失效
func InvalidateFollowingFeed(
	ctx context.Context,
	userID uint64,
) error {
	if database.Redis == nil {
		return nil
	}
	return database.Redis.Del(
		ctx,
		feedKey(userID),
		feedReadyKey(userID),
	).Err()
}
