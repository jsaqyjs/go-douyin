package main

import (
	"context"
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

	// 发布和消费使用两个独立Channel。
	pubCh, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer pubCh.Close()

	consumeCh, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer consumeCh.Close()

	if err := event_service.DeclareTopology(pubCh); err != nil {
		log.Fatal(err)
	}

	// 开启Publisher Confirm。
	if err := pubCh.Confirm(false); err != nil {
		log.Fatal(err)
	}

	confirmations := pubCh.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)

	// 每次只重放一条，便于人工控制。
	delivery, ok, err := consumeCh.Get(
		event_service.DeadQueue,
		false, // 手动ACK
	)
	if err != nil {
		log.Fatal(err)
	}
	if !ok {
		log.Println("死信队列中没有消息")
		return
	}

	// 保持原始业务事件ID不变。
	// 使用默认Exchange，路由到已声明的主队列。
	publish := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    delivery.MessageId,
		Body:         delivery.Body,
		Timestamp:    time.Now(),
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	err = pubCh.PublishWithContext(
		ctx,
		"", // 默认Exchange
		event_service.MainQueue,
		false,
		false,
		publish,
	)
	if err != nil {
		_ = delivery.Nack(false, true)
		log.Fatal(err)
	}

	// 等待发布者确认。
	select {
	case confirm, open := <-confirmations:
		if !open || !confirm.Ack {
			_ = delivery.Nack(false, true)
			log.Fatal("RabbitMQ未确认重放消息")
		}

	case <-ctx.Done():
		_ = delivery.Nack(false, true)
		log.Fatal("等待Publisher Confirm超时")
	}

	// 只有确认发布成功后，才ACK原死信。
	if err := delivery.Ack(false); err != nil {
		log.Fatal(err)
	}

	log.Println(
		"成功重放一条死信消息",
		"message_id:", delivery.MessageId,
	)
}
