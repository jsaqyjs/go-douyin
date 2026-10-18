package main

import (
	"context"
	"go-douyin/database"
	"go-douyin/service/outbox_service"
	"log"
	"log/slog"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 1. 初始化MySQL
	if err := database.InitDB(); err != nil {
		log.Fatal(err)
	}
	defer database.CloseMySQL()

	// 2. 监听停止信号
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	slog.Info("Outbox Dispatcher启动成功")

	// 每秒检查是否有待发送事件
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	// 3. 持续扫描
	for {
		select {
		case <-ctx.Done():
			slog.Info("Outbox Dispatcher退出")
			return

		case <-ticker.C:
			// 每轮最多处理100条，避免无上限占用资源
			for i := 0; i < 100; i++ {
				processed, err := outbox_service.DispatchOnce()
				if err != nil {
					slog.Error(
						"Outbox投递异常",
						"error", err,
					)
					// 出错后停止本轮，下一秒再试
					break
				}
				if !processed {
					// 没有待发送事件
					break
				}
			}
		}
	}
}
