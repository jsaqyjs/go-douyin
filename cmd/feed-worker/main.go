package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"go-douyin/database"
	"go-douyin/mq"
	"go-douyin/service/feed_service"
)

func main() {
	// 1. 初始化MySQL
	if err := database.InitDB(); err != nil {
		log.Fatal(err)
	}
	defer database.CloseMySQL()

	// 2. 初始化Redis
	if err := database.InitRedis(); err != nil {
		log.Fatal(err)
	}
	defer database.CloseRedis()

	// 3. 连接RabbitMQ
	conn, err := mq.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	// 4. 确认交换机和队列已创建
	if err := mq.DeclareTopology(ch); err != nil {
		log.Fatal(err)
	}

	// 每次最多分配1条未确认消息
	if err := ch.Qos(1, 0, false); err != nil {
		log.Fatal(err)
	}

	// 5. 注册消费者
	deliveries, ok := ch.Consume(
		mq.QueueName,
		"",
		false, // autoAck=false，必须手动确认
		false,
		false,
		false,
		nil,
	)
	if ok != nil {
		log.Fatal(ok)
	}
	// stop 是监控， ctx 是停止条件
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	slog.Info("Feed Worker已启动，等待消息")

	// 6. 消费消息
	for {
		select {
		case <-ctx.Done():
			slog.Info("Feed Worker准备退出")
			return

		case delivery, ok := <-deliveries:
			if !ok {
				slog.Error("RabbitMQ消息通道已关闭")
				return
			}

			err := processEvent(ctx, delivery.Body)

			if err != nil {
				slog.Error(
					"处理Feed事件失败",
					"error", err,
				)

				// 教学版：等待后重新入队
				// 正式版需要重试上限和死信队列
				time.Sleep(2 * time.Second)

				if err := delivery.Nack(
					false,
					true, // requeue
				); err != nil {
					slog.Error("Nack失败", "error", err)
					return
				}
				continue
			}

			// 7. 成功后才ACK
			if err := delivery.Ack(false); err != nil {
				slog.Error("ACK失败", "error", err)
				return
			}
		}
	}
}

// 处理视频发布事件
func processEvent(
	ctx context.Context,
	body []byte,
) error {
	var event mq.VideoPublishedEvent

	if err := json.Unmarshal(body, &event); err != nil {
		return err
	}

	if event.VideoID == 0 ||
		event.AuthorID == 0 ||
		event.CreatedAt.IsZero() {
		return errors.New("视频发布事件字段不合法")
	}

	// 给粉丝的Redis Feed分发视频
	err := feed_service.FanoutVideo(
		ctx,
		event.AuthorID,
		event.VideoID,
		event.CreatedAt,
	)
	if err != nil {
		return err
	}

	slog.Info(
		"视频Feed分发完成",
		"video_id", event.VideoID,
		"author_id", event.AuthorID,
	)

	return nil
}
