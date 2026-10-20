package feed_service

import (
	"context"
	"fmt"
	"time"

	"go-douyin/database"
)

const fanoutBatchSize = 100

// FanoutVideo 将视频推送给作者的粉丝
// 当前是同步、串行的实现
func FanoutVideo(
	ctx context.Context,
	authorID, videoID uint64,
	createdAt time.Time,
) error {
	var lastFollowID uint64

	for {
		// 查询一批粉丝
		type FollowRow struct {
			ID         uint64
			FollowerID uint64
		}

		var rows []FollowRow

		err := database.DB.WithContext(ctx).
			Table("follow").
			Select("id, follower_id").
			Where(
				"following_id = ? AND id > ?",
				authorID,
				lastFollowID,
			).
			Order("id ASC").
			Limit(fanoutBatchSize).
			Scan(&rows).Error

		if err != nil {
			return fmt.Errorf("查询粉丝失败: %w", err)
		}

		if len(rows) == 0 {
			break
		}

		// 写入每个粉丝的Feed收件箱
		for _, row := range rows {
			if err := PushVideoToFeed(
				ctx,
				row.FollowerID,
				videoID,
				createdAt,
			); err != nil {
				return fmt.Errorf(
					"推送粉丝%d失败: %w",
					row.FollowerID,
					err,
				)
			}
		}

		lastFollowID = rows[len(rows)-1].ID
	}

	return nil
}
