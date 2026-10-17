package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func InitRedis() error {
	addr := "127.0.0.1:6379"

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       0,
		PoolSize: 20,
	})

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fmt.Errorf("Redis连接失败: %w", err)
	}

	Redis = client
	return nil
}

func CloseRedis() error {
	if Redis == nil {
		return nil
	}
	return Redis.Close()
}
