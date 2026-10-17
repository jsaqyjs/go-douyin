package like_service

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go-douyin/database"
	"log/slog"
	"time"
)

const likeCacheTTL = 30 * time.Second

// 生成点赞状态缓存Key
func likeStatusKey(userID, videoID uint64) string {
	return fmt.Sprintf(
		"douyin:like:status:%d:%d",
		videoID,
		userID,
	)
}

// GetLikeCache 查询缓存
// 第一个返回值：是否点赞
// 第二个返回值：是否命中有效缓存
func GetLikeCache(userID, videoID uint64) (bool, bool) {
	if database.Redis == nil {
		return false, false
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()

	value, err := database.Redis.Get(
		ctx,
		likeStatusKey(userID, videoID),
	).Result()
	if err == redis.Nil {
		return false, false
	}

	if err != nil {
		slog.Warn("读取点赞缓存失败", "error", err)
		return false, false
	}

	switch value {
	case "1":
		return true, true
	case "0":
		return false, true
	default:
		return false, false
	}
}

//true, true	缓存命中，已经点赞
//false, true	缓存命中，没有点赞
//false, false	缓存未命中，需要查 MySQL

// SetLikeCache 写入缓存
func SetLikeCache(userID, videoID uint64, liked bool) {
	if database.Redis == nil {
		return
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()
	value := "0"
	if liked {
		value = "1"
	}
	err := database.Redis.Set(
		ctx,
		likeStatusKey(userID, videoID),
		value,
		likeCacheTTL,
	).Err()
	if err != nil {
		slog.Warn("写入点赞缓存失败", "error", err)
	}
}

// DeleteLikeCache 删除旧缓存
func DeleteLikeCache(userID, videoID uint64) {
	if database.Redis == nil {
		return
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()

	err := database.Redis.Del(
		ctx,
		likeStatusKey(userID, videoID),
	).Err()
	if err != nil {
		slog.Warn("删除点赞缓存失败", "error", err)
	}
}
