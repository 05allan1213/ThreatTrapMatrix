package mq_service

// File: honey_node/service/mq_service/mq_service.go
// Description: 节点MQ消费/生产核心模块，实现RabbitMQ队列/交换器声明、消息路由绑定、通用消费注册及队列消息发送能力，支撑创建IP、删除IP、端口绑定等指令的消费处理，以及告警/部署状态等消息的生产发送，保障节点与服务端的可靠通信

import (
	"encoding/json"
	"fmt"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Run 启动所有RabbitMQ消费者协程
func Run() {
	cfg := global.Config.MQ

	// 声明状态上报类队列
	queueDeclare(cfg.AlertTopic)
	queueDeclare(cfg.BatchDeployStatusTopic)
	queueDeclare(cfg.BatchUpdateDeployStatusTopic)
	queueDeclare(cfg.BatchRemoveDeployStatusTopic)

	// 注册业务指令交换器及消费回调
	go register(cfg.CreateIpExchangeName, CreateIpExChange)                   // 创建IP指令消费
	go register(cfg.DeleteIpExchangeName, DeleteIpExChange)                   // 删除IP指令消费
	go register(cfg.BindPortExchangeName, BindPortExChange)                   // 绑定端口指令消费
	go register(cfg.BatchDeployExchangeName, BatchDeployExChange)             // 批量部署指令消费
	go register(cfg.BatchUpdateDeployExchangeName, BatchUpdateDeployExChange) // 批量更新部署指令消费
	go register(cfg.BatchRemoveDeployExchangeName, BatchRemoveDeployExChange) // 批量删除部署指令消费

	// 监控检测
	go watchHealth()
}

// queueDeclare 声明RabbitMQ队列
func queueDeclare(queueName string) {
	_, err := global.Queue.QueueDeclare(
		queueName, // 队列名称
		true,      // 持久性：true表示队列数据持久化保存，MQ重启后数据不丢失
		false,     // 非自动删除：无消费者时不自动删除队列
		false,     // 非排他性：允许多个消费者监听同一队列
		false,     // 非阻塞：不等待服务器额外响应
		nil,       // 额外参数
	)
	if err != nil {
		logrus.Fatalf("声明队列失败%s %v", queueName, err) // 队列声明失败终止程序
		return
	}
	logrus.Infof("声明队列成功 %s", queueName)
}

// registerExchange 声明RabbitMQ交换器
func registerExchange(exchangeName string) {
	err := global.Queue.ExchangeDeclare(
		exchangeName, // 交换器名称
		"direct",     // 交换器类型：direct（按路由键精准路由到队列）
		true,         // 持久化：交换器在MQ重启后保留
		false,        // 非自动删除：无绑定队列时不自动删除
		false,        // 非内部交换器：允许外部生产者发送消息
		false,        // 非阻塞：不等待服务器额外响应
		nil,          // 额外参数
	)
	if err != nil {
		logrus.Fatalf("%s 声明交换器失败 %s", exchangeName, err) // 交换器声明失败终止程序
	}
	logrus.Infof("声明交换器成功 %s", exchangeName)
}

// register 通用MQ消费注册函数
func register[T any](exChangeName string, fun func(msg T) error) {
	// 声明业务指令交换器
	registerExchange(exChangeName)

	cf := global.Config

	// 构建节点专属队列名称（交换器名+节点UID），避免节点间队列冲突
	queueName := fmt.Sprintf("%s_%s_queue", exChangeName, cf.System.Uid)
	queueDeclare(queueName)

	// 绑定队列到交换器（路由键为节点UID，确保指令仅投递到当前节点）
	err := global.Queue.QueueBind(
		queueName,     // 待绑定的队列名称
		cf.System.Uid, // 绑定键（路由键）：与生产者侧的路由键一致（节点UID）
		exChangeName,  // 目标交换器名称
		false,         // 非阻塞：不等待服务器额外响应
		nil,           // 额外参数
	)
	if err != nil {
		logrus.Fatalf("%s 绑定队列失败 %s", queueName, err) // 队列绑定失败终止程序
	}

	// 启动消息消费（关闭自动确认，手动控制消息ACK/Nack）
	msgs, err := global.Queue.Consume(
		queueName, // 消费的队列名称
		"",        // 消费者标识：空字符串由MQ自动生成唯一标识
		false,     // 关闭自动确认：需手动调用Ack/Nack确认消息处理状态
		false,     // 非排他性：允许其他消费者消费同一队列（备用节点场景）
		false,     // 非本地消费者：接收所有投递到队列的消息
		false,     // 非阻塞：不等待服务器额外响应
		nil,       // 额外参数
	)
	if err != nil {
		logrus.Fatalf("%s 启动消费失败 %s", queueName, err) // 消费启动失败终止程序
	}

	// 循环消费消息
	for d := range msgs {
		var data T
		// 反序列化消息体为指定业务类型
		err = json.Unmarshal(d.Body, &data)
		if err != nil {
			logrus.Errorf("json解析失败 %s %s", err, string(d.Body))
			d.Ack(false) // 解析失败仍ACK，避免无效消息重复消费
			continue
		}

		// 调用业务回调处理消息
		err = fun(data)
		if err == nil {
			d.Ack(false) // 处理成功：手动确认消息（false表示仅确认当前消息）
			continue
		}
		// 处理失败：记录错误日志并确认消息
		d.Ack(false)
	}
	logrus.Errorf("%s 接收队列消息结束", queueName)
}

// sendQueueMessage 向指定队列发送消息（直接投递，无交换器）
func sendQueueMessage(queueName string, req any) (err error) {
	// 序列化业务请求为JSON字节数据（忽略序列化错误，仅捕获发送错误）
	byteData, _ := json.Marshal(req)

	// 直接向队列发送消息（交换器为空，路由键为队列名）
	err = global.Queue.Publish(
		"",        // 交换器：空字符串表示使用默认交换器（direct类型）
		queueName, // 路由键：目标队列名称，默认交换器会直接投递到该队列
		false,     // 强制投递：false表示队列不存在时丢弃消息
		false,     // 立即投递：false表示不强制立即投递到消费者
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 序列化后的业务数据
		})

	// 发送失败：记录错误日志（含队列名、错误信息、消息内容）
	if err != nil {
		logrus.Errorf("%s 发送消息失败: %v %s", queueName, err, string(byteData))
		return
	}

	// 发送成功：记录信息日志（含队列名、消息内容）
	logrus.Infof("%s 发送消息成功: %s", queueName, string(byteData))
	return nil
}

// watchHealth 监控MQ连接健康状态并实现自动重连
func watchHealth() {
	// 创建用于接收MQ连接关闭通知的通道
	closeQueue := make(chan *amqp.Error, 1)
	// 注册MQ连接关闭通知监听，当连接关闭时会向closeQueue发送关闭错误信息
	global.Queue.NotifyClose(closeQueue)

	// 启动独立协程处理MQ重连逻辑，避免阻塞主协程
	go func() {
		// 监听MQ关闭通知通道，接收连接关闭的错误信息
		for s := range closeQueue {
			fmt.Println("mq被关闭了", s)
		}
		// 打印通道关闭提示，进入重连等待阶段
		fmt.Println("通道被关闭, 等待重连")
		// 重连前延迟2秒，避免频繁重试占用资源
		time.Sleep(2 * time.Second)

		// 循环执行重连逻辑，直到重连成功
		for {
			// 尝试初始化MQ连接
			mq, err := core.InitMQ()
			if err != nil {
				// 重连失败，记录警告日志并延迟2秒后重试
				logrus.Warnf("mq连接失败, 等待重连 %s", err)
				time.Sleep(2 * time.Second)
				continue
			}
			// 重连成功，更新全局MQ实例
			global.Queue = mq
			// 重启MQ相关业务处理逻辑
			Run()
			// 重连成功，退出循环
			break
		}
	}()
}
