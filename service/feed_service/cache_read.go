package feed_service

import (
	"context"
	"fmt"
	"go-douyin/database"
	"go-douyin/dto"
	"go-douyin/models"
	"go-douyin/service/video_service"
	"go-douyin/utils"
	"strconv"
	"time"
)

// TryRedisFollowingFeed 尝试使用Redis查询第一页。
// hit=false表示需要回退到MySQL。
func TryRedisFollowingFeed(
	ctx context.Context,
	userID uint64,
	limit int,
) (dto.FollowingFeedResponse, bool, error) {
	result := dto.FollowingFeedResponse{
		List: make([]dto.VideoResponse, 0),
	}

	if database.Redis == nil {
		return result, false, nil
	}

	// 避免Redis故障导致HTTP请求长时间等待
	checkCtx, checkCancel := context.WithTimeout(
		ctx, 500*time.Millisecond,
	)

	// 1. 检查初始化标记
	ready, err := database.Redis.Exists(
		checkCtx,
		feedReadyKey(userID),
	).Result()
	checkCancel()
	if err != nil {
		return result, false, err
	}
	// 2. 未初始化则重建 重新从数据库中查找 放在redis中
	if ready == 0 {
		rebuildCtx, rebuildCancel := context.WithTimeout(
			ctx, 3*time.Second,
		)
		defer rebuildCancel()

		if err := RebuildFollowingFeed(
			rebuildCtx, userID,
		); err != nil {
			return result, false, err
		}
	}
	readCtx, readCancel := context.WithTimeout(
		ctx, 500*time.Millisecond,
	)
	// 3. 读取缓存中的候选视频ID
	members, err := database.Redis.ZRevRange(
		readCtx,
		feedKey(userID),
		0,
		feedMaxItems-1,
	).Result()
	defer readCancel()
	if err != nil {
		return result, false, err
	}

	if len(members) == 0 {
		// 空缓存可能是正常情况，也可能存在写入延迟。
		// 为保证数据可用，交给MySQL确认。
		return result, false, nil
	}
	ids := make([]uint64, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseUint(member, 10, 64)
		if err != nil {
			return result, false, err
		}
		ids = append(ids, id)
	}
	// 4. MySQL校验视频状态和当前关注关系
	var videos []models.Video

	err = database.DB.WithContext(ctx).
		Model(&models.Video{}).
		Joins(
			"JOIN follow AS f ON "+
				"f.following_id = video.author_id",
		).
		Where("f.follower_id = ?", userID).
		Where("video.id IN ?", ids).
		Where(
			"video.status = ?",
			models.VideoStatusPublished,
		).
		Select("video.*").
		Order("video.created_at DESC").
		Order("video.id DESC").
		Limit(limit + 1).
		Find(&videos).Error

	if err != nil {
		return result, false, fmt.Errorf(
			"校验Redis候选视频失败: %w", err,
		)
	}

	// 5. 候选数量不足，可能漏掉新视频或其他有效视频
	// 为保证这一页数量正确，回退MySQL完整查询
	if len(videos) < limit+1 {
		return result, false, nil
	}
	result.HasMore = true
	videos = videos[:limit]
	for _, video := range videos {
		result.List = append(
			result.List,
			video_service.ToVideoResponse(video),
		)
	}
	// 6. 使用MySQL排序后的最后一条数据生成游标
	last := videos[len(videos)-1]
	cursor, err := utils.EncodeFeedCursor(
		last.CreatedAt,
		last.ID,
	)
	if err != nil {
		return result, false, err
	}

	result.NextCursor = cursor

	return result, true, nil

}
