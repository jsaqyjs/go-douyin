package event_service

import (
	"context"
	"errors"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log/slog"
)

// ConsumeLikeEvents 阻塞处理点赞审计事件。
func ConsumeLikeEvents(
	ctx context.Context,
	conn *amqp.Connection,
) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := DeclareTopology(ch); err != nil {
		return err
	}

	// 每个消费者最多同时持有10条未ACK消息。
	if err := ch.Qos(10, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume(
		MainQueue,
		"",
		false, // autoAck=false，必须手动ACK
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("RabbitMQ消费通道已关闭")
			}

			if err := handleDelivery(ctx, delivery); err != nil {
				return err
			}
		}
	}
}
func handleDelivery(
	ctx context.Context,
	delivery amqp.Delivery,
) error {
	event, err := DecodeLikeEvent(delivery.Body)

	if err == nil {
		err = ProcessLikeEvent(ctx, event)
	}

	if err != nil {
		slog.Error(
			"点赞事件处理失败，进入死信队列",
			"event_id", delivery.MessageId,
			"error", err,
		)

		// 本课统一进入DLQ，等待检查后手动重放。
		if nackErr := delivery.Nack(false, false); nackErr != nil {
			return fmt.Errorf("NACK失败: %w", nackErr)
		}

		return nil
	}

	// 必须在MySQL事务成功提交之后ACK。
	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf("ACK失败: %w", err)
	}

	return nil
}
