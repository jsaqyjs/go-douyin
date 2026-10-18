package feed_service

import (
	"context"
	"testing"
	"time"

	"go-douyin/database"

	"github.com/redis/go-redis/v9"
)

func TestRedisFeed(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   0,
	})
	defer client.Close()

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis不可用: %v", err)
	}

	// 测试期间让业务函数使用这个Redis客户端
	oldClient := database.Redis
	database.Redis = client
	defer func() {
		database.Redis = oldClient
	}()

	// 使用专门的测试用户ID
	userID := uint64(999999)

	key := feedKey(userID)

	// 清空测试数据
	if err := client.Del(ctx, key).Err(); err != nil {
		t.Fatal(err)
	}
	defer client.Del(ctx, key)

	now := time.Now()

	// 模拟三个视频先后发布
	testVideos := []struct {
		id uint64
		at time.Time
	}{
		{101, now.Add(-3 * time.Minute)},
		{102, now.Add(-2 * time.Minute)},
		{103, now.Add(-1 * time.Minute)},
	}

	for _, video := range testVideos {
		err := PushVideoToFeed(
			ctx,
			userID,
			video.id,
			video.at,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 读取最新两个视频
	ids, err := ReadFeedIDs(ctx, userID, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 2 {
		t.Fatalf("期望2条，实际%d条", len(ids))
	}

	if ids[0] != 103 || ids[1] != 102 {
		t.Fatalf("排序错误: %v", ids)
	}

	t.Logf("Feed排序正确: %v", ids)
}
