package mq_service

// File: matrix_server/service/mq_service/enter.go
// Description: 初始化MQ相关资源，声明批量部署状态主题队列，并异步启动部署状态消息的消费协程

import (
	"encoding/json"
	"matrix_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Run 初始化MQ队列并启动消费协程
func Run() {
	// 获取全局配置中的MQ配置信息
	cfg := global.Config.MQ
	// 声明批量部署状态队列
	_, err := global.Queue.QueueDeclare(
		cfg.BatchDeployStatusTopic, // 队列名称：批量部署状态主题
		false,                      // 非持久化：队列不会在MQ重启后保留
		false,                      // 非自动删除：队列不会在所有消费者断开后自动删除
		false,                      // 非排他性：允许多个连接访问该队列
		false,                      // 非阻塞：声明队列操作不阻塞
		nil,                        // 额外队列参数（无）
	)
	if err != nil {
		logrus.Fatalf("声明队列失败: %v", err)
		return
	}
	_, err = global.Queue.QueueDeclare(
		cfg.BatchUpdateDeployStatusTopic, // 队列名称
		false,                            // 持久性
		false,                            // 自动删除
		false,                            // 排他性
		false,                            // 非阻塞
		nil,                              // 其他参数
	)
	if err != nil {
		logrus.Fatalf("声明队列失败: %v", err)
		return
	}
	_, err = global.Queue.QueueDeclare(
		cfg.BatchRemoveDeployStatusTopic, // 队列名称
		false,                            // 持久性
		false,                            // 自动删除
		false,                            // 排他性
		false,                            // 非阻塞
		nil,                              // 其他参数
	)
	if err != nil {
		logrus.Fatalf("声明队列失败: %v", err)
		return
	}

	// 注册交换器
	RegisterExChange()

	// 异步启动批量部署状态消息的消费协程
	go RevBatchDeployStatusMq()
	// 异步启动批量更新部署状态消息的消费协程
	go RevBatchUpdateDeployStatusMq()
	// 异步启动批量删除部署状态消息的消费协程
	go RevBatchRemoveDeployStatusMq()
}

// SendExchangeMessage 通用MQ消息发送函数
func SendExchangeMessage(exchangeName, nodeID string, req any) (err error) {
	// 将结构化请求数据序列化为JSON字节数据
	byteData, _ := json.Marshal(req)

	// 向RabbitMQ发布消息
	err = global.Queue.Publish(
		exchangeName, // 目标交换机名称
		nodeID,       // 路由键（节点ID）
		false,        // mandatory：是否强制要求消息路由到队列
		false,        // immediate：是否要求立即投递消息（
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型为纯文本（JSON格式）
			Body:        byteData,     // 消息体：JSON序列化后的请求数据
		})

	// 消息发布失败时记录错误日志
	if err != nil {
		logrus.Errorf("%s 消息发送失败 %s %s", exchangeName, err, string(byteData))
		return
	}

	// 消息发布成功时记录日志
	logrus.Infof("%s 消息发送成功 %s", exchangeName, string(byteData))
	return
}
