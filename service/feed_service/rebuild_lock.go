package feed_service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/redis/go-redis/v9"
	"go-douyin/database"
	"time"
)

// 相当于redis 锁的使用时间
const rebuildLockTTL = 10 * time.Second

var ErrRebuildBusy = errors.New("Feed正在被其他请求重建")

func feedLockKey(userID uint64) string {
	return feedKey(userID) + ":lock"
}

// acquireRebuildLock 尝试获取重建锁
func acquireRebuildLock(
	ctx context.Context,
	userID uint64,
) (string, bool, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", false, err
	}
	token := hex.EncodeToString(raw)

	ok, err := database.Redis.SetNX(
		ctx,
		feedLockKey(userID),
		token,
		rebuildLockTTL,
	).Result()
	return token, ok, err
}

// 仅在锁仍属于自己时释放
var unlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
end
return 0
`)

func releaseRebuildLock(
	ctx context.Context,
	userID uint64,
	token string,
) error {
	return unlockScript.Run(
		ctx,
		database.Redis,
		[]string{feedLockKey(userID)},
		token,
	).Err()
}

// EnsureFollowingFeedCache 只允许一个请求执行重建。
// 如果其他请求正在重建，返回 ErrRebuildBusy，
// 上层可以直接使用MySQL查询。
func EnsureFollowingFeedCache(
	ctx context.Context,
	userID uint64,
) error {
	if database.Redis == nil {
		return errors.New("Redis不可用")
	}

	token, acquired, err := acquireRebuildLock(
		ctx, userID,
	)
	if err != nil {
		return err
	}

	if !acquired {
		return ErrRebuildBusy
	}
	// 使用独立的短超时释放锁，
	// 避免请求Context已取消时无法解锁。
	defer func() {
		unlockCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Second,
		)
		defer cancel()

		_ = releaseRebuildLock(
			unlockCtx, userID, token,
		)
	}()
	// 获得锁后再次检查ready，
	// 防止其他请求已完成重建。
	ready, err := database.Redis.Exists(
		ctx,
		feedReadyKey(userID),
	).Result()
	if err != nil {
		return err
	}

	if ready > 0 {
		return nil
	}
	// 只有锁持有者执行重建
	return RebuildFollowingFeed(ctx, userID)
}
