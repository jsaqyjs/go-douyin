package like_service

import (
	"context"
	"testing"

	"go-douyin/database"

	"github.com/redis/go-redis/v9"
)

func TestDemoRedisLike(t *testing.T) {
	database.Redis = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	defer database.Redis.Close()

	// 使用独立测试视频ID
	videoID := uint64(999999)

	keys := []string{
		likeDemoUsersKey(videoID),
		likeDemoCountKey(videoID),
	}

	// 测试前清理测试Key
	ctx := context.Background()
	if err := database.Redis.Del(ctx, keys...).Err(); err != nil {
		t.Fatal(err)
	}
	defer database.Redis.Del(ctx, keys...)

	// 第一次点赞
	result1, err := DemoRedisLike(5, videoID)
	if err != nil {
		t.Fatal(err)
	}

	// 第二次点赞同一视频
	result2, err := DemoRedisLike(5, videoID)
	if err != nil {
		t.Fatal(err)
	}

	// 第二个用户点赞
	result3, err := DemoRedisLike(10, videoID)
	if err != nil {
		t.Fatal(err)
	}

	if result1 != 1 || result2 != 0 || result3 != 1 {
		t.Fatalf(
			"结果不符合预期: %d, %d, %d",
			result1, result2, result3,
		)
	}

	count, err := database.Redis.Get(
		ctx,
		likeDemoCountKey(videoID),
	).Int64()

	if err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Fatalf("期望点赞数为2，实际为%d", count)
	}
}
