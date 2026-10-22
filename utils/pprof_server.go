package utils

import (
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
)

// StartLocalPprof 启动本地专用的性能分析端口。
// 只能在开发环境中调用。
func StartLocalPprof() {
	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:6060",
	)
	if err != nil {
		slog.Error("pprof监听失败", "error", err)
		return
	}

	go func() {
		slog.Info("pprof已启动",
			"addr", "127.0.0.1:6060",
		)

		if err := http.Serve(
			listener,
			http.DefaultServeMux,
		); err != nil {
			slog.Error("pprof服务退出", "error", err)
		}
	}()
}
