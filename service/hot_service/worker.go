package hot_service

import (
	"context"
	"log/slog"
	"time"
)

// RunHotWorker 定时更新热门视频排行榜
func RunHotWorker(ctx context.Context) {
	// 启动时先构建一次
	if err := RebuildHotRank(ctx); err != nil {
		slog.Error("首次构建热榜失败",
			"error", err,
		)
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("热门榜Worker退出")
			return

		case <-ticker.C:
			if err := RebuildHotRank(ctx); err != nil {
				slog.Error("刷新热门榜失败",
					"error", err,
				)
				continue
			}

			slog.Info("热门视频排行榜刷新成功")
		}
	}
}
