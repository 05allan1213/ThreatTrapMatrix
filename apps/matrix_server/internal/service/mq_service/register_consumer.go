package mq_service

// File: matrix_server/service/mq_service/register_consumer.go
// Description: MQ通用消费者注册模块，基于泛型实现通用的消息消费逻辑，自动完成消息队列监听、JSON反序列化及自定义处理函数调用，适配任意结构化消息格式

import (
	"encoding/json"
	"log"
	"matrix_server/internal/global"

	"github.com/sirupsen/logrus"
)

// registerConsumer 通用MQ消费者注册函数
func registerConsumer[T any](queueName string, fun func(msg T)) {
	// 注册MQ消费者，监听指定队列
	msgs, err := global.Queue.Consume(
		queueName, // 消费的目标队列名称
		"",        // 消费者标识（空表示由MQ自动分配）
		true,      // 自动确认消息（消费后无需手动ACK，MQ直接标记为已处理）
		false,     // 排他性（false表示非排他消费，多个消费者可同时消费该队列）
		false,     // 非本地（false表示接收本地发布的消息）
		false,     // 非阻塞（false表示阻塞等待消息，有消息时立即处理）
		nil,       // 额外配置参数（无特殊配置传nil）
	)
	if err != nil {
		// 消费者注册失败时终止程序
		log.Fatalf("无法注册消费者: %v", err)
	}

	// 循环监听并处理队列消息
	for d := range msgs {
		// 初始化泛型类型变量，用于接收反序列化后的消息数据
		var data T
		// 将消息体JSON数据反序列化为泛型T类型
		err = json.Unmarshal(d.Body, &data)
		if err != nil {
			// 反序列化失败时记录错误日志，跳过当前消息
			logrus.Errorf("json解析失败 %s %s", err, string(d.Body))
			continue
		}
		// 记录消息接收日志（含队列名和消息内容）
		logrus.Infof("%s 接收回调消息 %s", queueName, string(d.Body))
		// 调用自定义处理函数处理解析后的消息数据
		fun(data)
	}
}
