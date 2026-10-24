package main

import (
	"context"
	"go-douyin/database"
	"go-douyin/mq"
	"log"
	"time"

	"go-douyin/service/event_service"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {

	conn, err := mq.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	// 1. 初始化 MySQL
	if err := database.InitDB(); err != nil {
		log.Fatal(err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	if err := event_service.DeclareTopology(ch); err != nil {
		log.Fatal(err)
	}

	if err := ch.Confirm(false); err != nil {
		log.Fatal(err)
	}

	confirms := ch.NotifyPublish(
		make(chan amqp.Confirmation, 2),
	)

	body := []byte(`{
        "event_id":"like-test-001",
        "video_id":15,
        "user_id":3
    }`)

	// 故意连续发送两次完全相同的事件。
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)

		err := ch.PublishWithContext(
			ctx,
			event_service.MainExchange,
			event_service.MainKey,
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				MessageId:    "like-test-001",
				Body:         body,
			},
		)

		if err == nil {
			select {
			case confirm, ok := <-confirms:
				if !ok || !confirm.Ack {
					log.Fatal("消息未被确认")
				}
			case <-ctx.Done():
				log.Fatal("等待确认超时")
			}
		}

		cancel()

		if err != nil {
			log.Fatal(err)
		}

		log.Printf("第%d次发送成功", i+1)
	}
}
