package mq

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ConfirmPublisher 负责串行发布并等待确认。
// 一个实例不可被多个goroutine同时调用。
type ConfirmPublisher struct {
	ch       *amqp.Channel
	confirms <-chan amqp.Confirmation
}

func NewConfirmPublisher(
	ch *amqp.Channel,
) (*ConfirmPublisher, error) {
	if err := ch.Confirm(false); err != nil {
		return nil, err
	}

	confirms := ch.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)

	return &ConfirmPublisher{
		ch:       ch,
		confirms: confirms,
	}, nil
}

func (p *ConfirmPublisher) Publish(
	routingKey string,
	message amqp.Publishing,
) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := p.ch.PublishWithContext(
		ctx,
		ExchangeName,
		routingKey,
		false,
		false,
		message,
	); err != nil {
		return err
	}

	select {
	case confirm, ok := <-p.confirms:
		if !ok || !confirm.Ack {
			return errors.New("消息发布未获确认")
		}
		return nil

	case <-ctx.Done():
		return fmt.Errorf(
			"等待Publisher Confirm超时: %w",
			ctx.Err(),
		)
	}
}
