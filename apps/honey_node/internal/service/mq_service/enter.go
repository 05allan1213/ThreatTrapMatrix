package mq_service

// File: honey_node/service/mq_service/mq_service.go
// Description: RabbitMQ消费者服务，负责注册不同业务交换器的消费者，处理消息消费与确认逻辑

import (
	"fmt"
	"honey_node/internal/global"

	"github.com/sirupsen/logrus"
)

// Run 启动所有RabbitMQ消费者协程
func Run() {
	cfg := global.Config.MQ
	// 声明队列
	_, err := global.Queue.QueueDeclare(
		cfg.AlertTopic, // 队列名称
		false,          // 持久性
		false,          // 自动删除
		false,          // 排他性
		false,          // 非阻塞
		nil,            // 其他参数
	)
	if err != nil {
		logrus.Fatalf("AlertTopic声明队列失败: %v", err)
		return
	}
	_, err = global.Queue.QueueDeclare(
		cfg.BatchDeployStatusTopic, // 队列名称
		false,                      // 持久性
		false,                      // 自动删除
		false,                      // 排他性
		false,                      // 非阻塞
		nil,                        // 其他参数
	)
	if err != nil {
		logrus.Fatalf("BatchDeployStatusTopic声明队列失败: %v", err)
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
		logrus.Fatalf("BatchUpdateDeployStatusTopic声明队列失败: %v", err)
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
		logrus.Fatalf("BatchRemoveDeployStatusTopic声明队列失败: %v", err)
		return
	}

	// 启动创建IP交换器的消费者协程
	go register(cfg.CreateIpExchangeName, CreateIpExChange)
	// 启动删除IP交换器的消费者协程
	go register(cfg.DeleteIpExchangeName, DeleteIpExChange)
	// 启动绑定端口交换器的消费者协程
	go register(cfg.BindPortExchangeName, BindPortExChange)
	// 启动批量部署交换器的消费者协程
	go register(cfg.BatchDeployExchangeName, BatchDeployExChange)
	// 启动批量更新部署交换器的消费者协程
	go register(cfg.BatchUpdateDeployExchangeName, BatchUpdateDeployExChange)
	// 启动批量删除部署交换器的消费者协程
	go register(cfg.BatchRemoveDeployExchangeName, BatchRemoveDeployExChange)
}

// register 注册单个交换器的消费者逻辑
func register(exChangeName string, fun func(msg string) error) {
	// 声明交换器（确保与生产者交换器一致，防止生产者未提前声明）
	err := global.Queue.ExchangeDeclare(
		exChangeName, // 交换器名称（与生产者保持一致）
		"direct",     // 交换器类型：direct（直接匹配路由键）
		true,         // 持久化：交换器重启后保留
		false,        // 自动删除：不自动删除
		false,        // 内部交换器：否（允许客户端发送消息）
		false,        // 非阻塞：立即返回
		nil,          // 额外参数：无
	)
	if err != nil {
		logrus.Fatalf("%s 声明交换器失败 %s", exChangeName, err)
	}

	cf := global.Config
	// 声明专属队列（按节点UID命名，确保队列唯一性）
	queue, err := global.Queue.QueueDeclare(
		fmt.Sprintf("%s_%s_queue", exChangeName, cf.System.Uid), // 队列名称（唯一标识，与node01绑定）
		true,                                                    // 持久化队列：队列重启后保留
		false,                                                   // 自动删除：不自动删除
		false,                                                   // 排他性：否（允许多个消费者消费，但此处每个节点独占队列）
		false,                                                   // 非阻塞：立即返回
		nil,                                                     // 额外参数：无
	)
	if err != nil {
		logrus.Fatalf("声明队列失败 %s", err)
	}

	// 将队列绑定到交换器，指定路由键为节点UID（确保消息精准路由到当前节点）
	err = global.Queue.QueueBind(
		queue.Name,    // 队列名称
		cf.System.Uid, // 路由键：节点唯一标识，与生产者发送时的路由键匹配
		exChangeName,  // 交换器名称
		false,         // 非阻塞：立即返回
		nil,           // 额外参数：无
	)
	if err != nil {
		logrus.Fatalf("%s 绑定队列失败 %s", queue.Name, err)
	}

	// 注册消费者，开始监听队列消息（关闭自动确认，手动控制消息确认）
	msgs, err := global.Queue.Consume(
		queue.Name, // 要消费的队列名称
		"",         // 消费者标签：空（使用默认标识）
		false,      // 自动确认：否（手动Ack/Nack）
		false,      // 排他性：否
		false,      // 本地消费者：否（接收所有节点发送的消息）
		false,      // 非阻塞：立即返回
		nil,        // 额外参数：无
	)

	logrus.Infof("绑定交换器成功 %s", exChangeName)

	// 循环消费消息通道中的消息
	for d := range msgs {
		// 调用业务处理函数处理消息内容
		err = fun(string(d.Body))
		if err == nil {
			// 处理成功：手动确认消息（false表示仅确认当前消息）
			d.Ack(false)
			continue
		}
		// 处理失败：拒绝消息并重新入队（false不批量，true重新入队）
		d.Nack(false, true)
	}
}
