package feed_service

import (
	"context"
	"go-douyin/database"
	"strconv"
	"testing"
)

func benchmarkFeed(b *testing.B, useRedis bool) {
	userIDText := "1"
	userID, err := strconv.ParseUint(
		userIDText, 10, 64,
	)
	if err != nil || userID == 0 {
		b.Fatal("BENCH_USER_ID不合法")
	}

	if err := database.InitDB(); err != nil {
		b.Fatal(err)
	}
	defer database.CloseMySQL()
	if useRedis {
		if err := database.InitRedis(); err != nil {
			b.Fatal(err)
		}
		defer database.CloseRedis()

		// 预热缓存，尽量测量命中时的表现
		ctx := context.Background()
		if err := EnsureFollowingFeedCache(
			ctx, userID,
		); err != nil {
			b.Fatal(err)
		}
	}
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if useRedis {
			_, err = ListFollowingFeed(
				ctx, userID, 10, "",
			)
		} else {
			_, err = listFollowingFeedFromMySQL(
				ctx, userID, 10, "",
			)
		}

		if err != nil {
			b.Fatal(err)
		}
	}

}
func BenchmarkMySQLFeed(b *testing.B) {
	benchmarkFeed(b, false)
}

func BenchmarkRedisFeed(b *testing.B) {
	benchmarkFeed(b, true)
}
