package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "douyin.events"
	QueueName    = "douyin.feed.fanout"
	RoutingKey   = "video.published"
)

// 一条视频发布事件
type VideoPublishedEvent struct {
	VideoID   uint64    `json:"video_id"`
	AuthorID  uint64    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

func Connect() (*amqp.Connection, error) {
	url := "amqp://admin:admin@localhost:5672/"
	return amqp.Dial(url)
}

// 声明交换机、队列和绑定关系
func DeclareTopology(ch *amqp.Channel) error {
	err := ch.ExchangeDeclare(
		ExchangeName,
		"direct",
		true,  // durable
		false, // autoDelete
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		QueueName,
		true, // durable
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return ch.QueueBind(
		QueueName,
		RoutingKey,
		ExchangeName,
		false,
		nil,
	)
}

// 发布事件，并等待RabbitMQ确认
func PublishVideoPublished(
	event VideoPublishedEvent,
) error {
	conn, err := Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := DeclareTopology(ch); err != nil {
		return err
	}

	if err := ch.Confirm(false); err != nil {
		return err
	}

	confirms := ch.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	err = ch.PublishWithContext(
		ctx,
		ExchangeName,
		RoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return err
	}

	select {
	case confirm, ok := <-confirms:
		if !ok || !confirm.Ack {
			return errors.New("RabbitMQ未确认消息")
		}
		return nil

	case <-ctx.Done():
		return fmt.Errorf(
			"等待消息确认超时: %w",
			ctx.Err(),
		)
	}
}
