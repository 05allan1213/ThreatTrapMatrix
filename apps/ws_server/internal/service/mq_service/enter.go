package mq_service

// File: ws_server/service/mq_service/enter.go
// Description: MQ服务初始化模块，负责MQ队列声明、交换器注册、消费者协程启动，完成MQ消费服务的整体初始化，支撑批量部署/更新/删除的状态消息消费

import (
	"ws_server/internal/global"

	"github.com/sirupsen/logrus"
)

// Run MQ核心资源初始化：声明业务队列、注册交换器、启动各队列的消费协程，支撑批量部署/更新/删除的状态消息消费
func Run() {
	// 获取全局MQ配置
	cfg := global.Config.MQ

	// 声明业务所需的MQ队列
	queueDeclare(cfg.WsTopic)

	// 异步启动各队列的消费者协程
	go registerConsumer(cfg.WsTopic, wsConsumer)
}

// queueDeclare 声明MQ队列
func queueDeclare(queueName string) {
	_, err := global.Queue.QueueDeclare(
		queueName, // 队列名称
		false,     // 持久性（false表示队列非持久化，服务重启后队列消失）
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
