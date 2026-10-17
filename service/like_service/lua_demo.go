package like_service

import (
	"context"
	"strconv"
	"time"

	"go-douyin/database"

	"github.com/redis/go-redis/v9"
)

// 教学示例：仅操作演示专用Redis Key
var demoLikeScript = redis.NewScript(`
    local added = redis.call(
        "SADD", KEYS[1], ARGV[1]
    )

    if added == 0 then
        return 0
    end

    redis.call("INCR", KEYS[2])
    return 1
`)

func DemoRedisLike(userID, videoID uint64) (int64, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	keys := []string{
		likeDemoUsersKey(videoID),
		likeDemoCountKey(videoID),
	}

	return demoLikeScript.Run(
		ctx,
		database.Redis,
		keys,
		userID,
	).Int64()
}

func likeDemoUsersKey(videoID uint64) string {
	return "douyin:demo:likers:" + uintToString(videoID)
}

func likeDemoCountKey(videoID uint64) string {
	return "douyin:demo:count:" + uintToString(videoID)
}
func uintToString(id uint64) string {
	return strconv.FormatUint(id, 10)
}
