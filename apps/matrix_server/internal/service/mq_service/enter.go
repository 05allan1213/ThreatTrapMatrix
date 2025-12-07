package mq_service

// File: matrix_server/service/mq_service/enter.go
// Description: MQ服务初始化模块，负责MQ队列声明、交换器注册、消费者协程启动，完成MQ消费服务的整体初始化，支撑批量部署/更新/删除的状态消息消费

import (
	"encoding/json"
	"matrix_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Run MQ核心资源初始化：声明业务队列、注册交换器、启动各队列的消费协程，支撑批量部署/更新/删除的状态消息消费
func Run() {
	// 获取全局MQ配置
	cfg := global.Config.MQ

	// 声明业务所需的MQ队列（确保队列存在，消费前完成初始化）
	queueDeclare(cfg.BatchDeployStatusTopic)       // 批量部署状态反馈队列
	queueDeclare(cfg.BatchUpdateDeployStatusTopic) // 批量更新部署状态反馈队列
	queueDeclare(cfg.BatchRemoveDeployStatusTopic) // 批量删除部署状态反馈队列

	// 注册MQ交换器（定义消息路由规则）
	RegisterExChange()

	// 异步启动各队列的消费者协程
	go registerConsumer(cfg.BatchDeployStatusTopic, revBatchDeployStatusMq)
	go registerConsumer(cfg.BatchUpdateDeployStatusTopic, revBatchUpdateDeployStatusMq)
	go registerConsumer(cfg.BatchRemoveDeployStatusTopic, revBatchRemoveDeployStatusMq)
}

// queueDeclare 声明MQ队列
func queueDeclare(queueName string) {
	_, err := global.Queue.QueueDeclare(
		queueName, // 队列名称
		true,      // 持久性：(true表示队列数据持久化保存，MQ重启后数据不丢失)
		false,     // 自动删除（false表示队列不会自动删除）
		false,     // 排他性（false表示非排他队列，多个消费者可连接）
		false,     // 非阻塞（false表示阻塞等待队列声明完成）
		nil,       // 额外配置参数（无特殊配置传nil）
	)
	if err != nil {
		// 队列声明失败为致命错误，终止程序
		logrus.Fatalf("声明队列失败: %v", err)
		return
	}
	// 队列声明成功记录日志
	logrus.Infof("%s 声明队列成功", queueName)
}

// sendExchangeMessage 通用MQ消息发送函数
func sendExchangeMessage(exchangeName, nodeID string, req any) (err error) {
	// 将结构化请求数据序列化为JSON字节数据（忽略序列化错误，保持原有逻辑）
	byteData, _ := json.Marshal(req)

	// 向RabbitMQ发布消息
	err = global.Queue.Publish(
		exchangeName, // 目标交换器名称
		nodeID,       // 路由键（节点ID）
		false,        // mandatory：是否强制要求消息路由到队列（false表示不强制）
		false,        // immediate：是否要求立即投递消息（false表示不立即）
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
