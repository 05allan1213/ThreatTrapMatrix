package mq_service

// File: ws_server/service/mq_service/register_consumer.go
// Description: MQ通用消费者注册模块，基于泛型实现通用的消息消费逻辑，自动完成消息队列监听、JSON反序列化及自定义处理函数调用，适配任意结构化消息格式

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"runtime/debug"
	"unicode/utf8"
	"ws_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// registerConsumer 通用MQ消费者注册函数
func registerConsumer[T any](queueName string, handler func(msg T)) {
	msgs, err := global.Queue.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to register consumer: %v", err)
	}

	// 循环监听并处理队列消息
	for delivery := range msgs {
		if !handleMessage(queueName, delivery, handler) {
			continue
		}
	}
}

const bodyPreviewLimit = 1024

// handleMessage 单条消息处理入口，负责反序列化、异常恢复与日志
func handleMessage[T any](queueName string, delivery amqp.Delivery, handler func(msg T)) (ok bool) {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			logrus.WithFields(logrus.Fields{
				"queue_name":     queueName,
				"panic":          panicValue,
				"stack":          string(debug.Stack()),
				"delivery_tag":   delivery.DeliveryTag,
				"exchange":       delivery.Exchange,
				"routing_key":    delivery.RoutingKey,
				"message_id":     delivery.MessageId,
				"correlation_id": delivery.CorrelationId,
			}).Error("mq consumer panic recovered")
			ok = false
		}
	}()

	var data T
	if err := json.Unmarshal(delivery.Body, &data); err != nil {
		// 反序列化失败直接记录并跳过，保证后续消息继续消费
		preview, encoding := bodyPreview(delivery.Body)
		logrus.WithFields(logrus.Fields{
			"queue_name":            queueName,
			"body_len":              len(delivery.Body),
			"body_preview":          preview,
			"body_preview_encoding": encoding,
			"delivery_tag":          delivery.DeliveryTag,
			"exchange":              delivery.Exchange,
			"routing_key":           delivery.RoutingKey,
			"message_id":            delivery.MessageId,
			"correlation_id":        delivery.CorrelationId,
		}).WithError(err).Error("json unmarshal failed")
		return false
	}

	// 业务处理函数
	handler(data)
	return true
}

// bodyPreview 截断并安全输出消息体预览，避免日志爆炸
func bodyPreview(body []byte) (string, string) {
	if len(body) == 0 {
		return "", "empty"
	}
	limit := bodyPreviewLimit
	if len(body) < limit {
		limit = len(body)
	}
	// 仅截取前 N 字节作为预览
	chunk := body[:limit]
	if utf8.Valid(chunk) {
		return string(chunk), "utf8"
	}
	// 非 UTF-8 时输出 hex 预览
	return hex.EncodeToString(chunk), "hex"
}
