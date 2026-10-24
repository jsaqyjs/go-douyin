package main

import (
	"context"
	"go-douyin/database"
	"go-douyin/mq"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-douyin/service/event_service"
)

func main() {

	conn, err := mq.Connect()
	if err != nil {
		log.Fatal(err)
	}
	// 1. 初始化 MySQL
	if err := database.InitDB(); err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Println("点赞事件消费者已启动")

	if err := event_service.ConsumeLikeEvents(
		ctx, conn,
	); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
	log.Println("结束")
}
