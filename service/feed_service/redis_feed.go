package feed_service

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"go-douyin/service/follow_service"
	"go-douyin/service/video_service"
	"strconv"
	"time"
)

const (
	feedMaxItems = 1000
	feedTTL      = 7 * 24 * time.Hour
)

// 用户的Redis Feed Key
func feedKey(userID uint64) string {
	return fmt.Sprintf(
		"douyin:feed:following:%d",
		userID,
	)
}

// PushVideoToFeed 将视频放入某个用户的Feed
func PushVideoToFeed(
	ctx context.Context,
	userID, videoID uint64,
	createdAt time.Time,
) error {
	if database.Redis == nil {
		return fmt.Errorf("Redis尚未初始化")
	}
	key := feedKey(userID)
	// 毫秒时间戳作为排序分数
	score := float64(createdAt.UnixMilli())
	err := database.Redis.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: strconv.FormatUint(videoID, 10),
	}).Err()
	if err != nil {
		return err
	}

	// 2. 仅保留最新1000条
	// 按Score升序排列，删除较旧的成员
	count, err := database.Redis.ZCard(ctx, key).Result()
	if err != nil {
		return err
	}

	if count > feedMaxItems {
		err = database.Redis.ZRemRangeByRank(ctx, key,
			0, count-feedMaxItems-1).Err()

		if err != nil {
			return err
		}
	}
	// 3. 设置过期时间
	return database.Redis.Expire(ctx, key, feedTTL).Err()
}

// ReadFeedIDs 从Redis读取最新视频ID
func ReadFeedIDs(
	ctx context.Context,
	userID uint64,
	limit int64,
) ([]uint64, error) {
	if database.Redis == nil {
		return nil, fmt.Errorf("Redis尚未初始化")
	}

	if limit < 1 || limit > 50 {
		return nil, follow_service.ErrInvalidLimit
	}
	members, err := database.Redis.ZRevRange(ctx, feedKey(userID), 0,
		limit-1,
	).Result()
	if err != nil {
		return nil, err
	}

	ids := make([]uint64, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseUint(member, 10, 64)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}

// LoadFeedVideos 根据Redis中的视频ID读取完整视频信息
func LoadFeedVideos(
	ctx context.Context,
	ids []uint64,
) ([]dto.VideoResponse, error) {
	result := make([]dto.VideoResponse, 0, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var videos []models.Video
	err := database.DB.WithContext(ctx).
		Where("id IN ? AND status = ?",
			ids,
			models.VideoStatusPublished,
		).
		Find(&videos).Error

	if err != nil {
		return nil, err
	}

	// 建立 ID -> Video 的映射
	videoMap := make(map[uint64]models.Video, len(videos))
	for _, video := range videos {
		videoMap[video.ID] = video
	}

	// 按Redis返回的ID顺序组装结果
	for _, id := range ids {
		video, exists := videoMap[id]
		if !exists {
			// 可能已删除或下架
			continue
		}

		result = append(
			result,
			video_service.ToVideoResponse(video),
		)
	}

	return result, nil
}
