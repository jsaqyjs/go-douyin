package event_service

import amqp "github.com/rabbitmq/amqp091-go"

const (
	MainExchange = "douyin.events"
	MainQueue    = "douyin.like.audit.q"
	MainKey      = "video.liked.audit"

	DeadExchange = "douyin.dlx"
	DeadQueue    = "douyin.like.audit.dlq"
	DeadKey      = "video.liked.failed"
)

// 创建队列 交换机 路由
func DeclareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		MainExchange,
		"direct",
		true, false, false, false, nil,
	); err != nil {
		return err
	}

	if err := ch.ExchangeDeclare(
		DeadExchange,
		"direct",
		true, false, false, false, nil,
	); err != nil {
		return err
	}

	// 死信队列
	if _, err := ch.QueueDeclare(
		DeadQueue,
		true, false, false, false, nil,
	); err != nil {
		return err
	}

	if err := ch.QueueBind(
		DeadQueue,
		DeadKey,
		DeadExchange,
		false, nil,
	); err != nil {
		return err
	}

	// 主队列配置死信交换机。
	args := amqp.Table{
		"x-dead-letter-exchange":    DeadExchange,
		"x-dead-letter-routing-key": DeadKey,
	}

	if _, err := ch.QueueDeclare(
		MainQueue,
		true, false, false, false, args,
	); err != nil {
		return err
	}

	return ch.QueueBind(
		MainQueue,
		MainKey,
		MainExchange,
		false, nil,
	)
}
