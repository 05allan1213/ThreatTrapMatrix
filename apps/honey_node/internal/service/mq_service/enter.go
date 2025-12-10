package mq_service

// File: honey_node/service/mq_service/mq_service.go
// Description: MQ服务核心模块，负责RabbitMQ队列/交换器的声明与绑定、业务消费端注册、消息发送、连接健康监控及自动重连，封装不同业务场景的MQ消费逻辑

import (
	"encoding/json"
	"fmt"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// wg 同步等待组
// 用于等待所有业务交换器的消费端注册完成，确保首次初始化时所有消费逻辑都已就绪
var wg = sync.WaitGroup{}

// Run MQ服务启动入口函数
func Run() {
	cfg := global.Config.MQ

	// 首次初始化时，声明状态通知类队列（供消费端接收部署/更新/移除状态）
	if !cfg.InitMQ {
		queueDeclare(cfg.AlertTopic)                   // 告警通知队列
		queueDeclare(cfg.BatchDeployStatusTopic)       // 批量部署状态队列
		queueDeclare(cfg.BatchUpdateDeployStatusTopic) // 批量更新部署状态队列
		queueDeclare(cfg.BatchRemoveDeployStatusTopic) // 批量移除部署状态队列
	}

	// 启动独立协程处理消费端注册，避免阻塞主协程
	go func() {
		// 注册6个核心业务交换器的消费端，WaitGroup计数+6
		wg.Add(6)

		// 注册IP创建业务消费端，间隔200ms避免并发声明资源冲突
		go register(cfg.CreateIpExchangeName, CreateIpExChange)
		time.Sleep(200 * time.Millisecond)
		// 注册IP删除业务消费端
		go register(cfg.DeleteIpExchangeName, DeleteIpExChange)
		time.Sleep(200 * time.Millisecond)
		// 注册端口绑定业务消费端
		go register(cfg.BindPortExchangeName, BindPortExChange)
		time.Sleep(200 * time.Millisecond)
		// 注册批量部署业务消费端
		go register(cfg.BatchDeployExchangeName, BatchDeployExChange)
		time.Sleep(200 * time.Millisecond)
		// 注册批量更新部署业务消费端
		go register(cfg.BatchUpdateDeployExchangeName, BatchUpdateDeployExChange)
		time.Sleep(200 * time.Millisecond)
		// 注册批量移除部署业务消费端
		go register(cfg.BatchRemoveDeployExchangeName, BatchRemoveDeployExChange)

		// 等待所有消费端注册完成
		wg.Wait()

		// 首次初始化完成后，更新配置标记并持久化，避免重复声明队列/交换器
		if !global.Config.MQ.InitMQ {
			global.Config.MQ.InitMQ = true
			core.SetConfig(global.Config)
			logrus.Infof("全部交换器注册及绑定完成")
		}
	}()

	// 启动MQ连接健康监控，异常时自动重连
	go watchHealth()
}

// queueDeclare 声明RabbitMQ队列
func queueDeclare(queueName string) {
	_, err := global.Queue.QueueDeclare(
		queueName, // 队列名称
		true,      // 持久化：队列元数据持久化到磁盘，服务重启不丢失
		false,     // 自动删除：关闭最后一个消费者后不自动删除队列
		false,     // 排他性：非排他队列，允许多个消费者连接
		false,     // 非阻塞：同步声明队列，等待结果返回
		nil,       // 额外参数：无特殊配置
	)
	if err != nil {
		logrus.Fatalf("声明队列失败%s %v", queueName, err)
		return
	}
	logrus.Infof("声明队列成功 %s", queueName)
}

// registerExchange 声明RabbitMQ交换器
func registerExchange(exchangeName string) {
	err := global.Queue.ExchangeDeclare(
		exchangeName, // 交换器名称（需与生产者端一致）
		"direct",     // 交换器类型：direct（直接交换器），按路由键精准匹配队列
		true,         // 持久化：交换器元数据持久化，服务重启不丢失
		false,        // 自动删除：无绑定队列时不自动删除交换器
		false,        // 内部交换器：非内部，允许生产者发送消息
		false,        // 非阻塞：同步声明交换器，等待结果返回
		nil,          // 额外参数：无特殊配置
	)
	if err != nil {
		logrus.Fatalf("%s 声明交换器失败 %s", exchangeName, err)
	}
	logrus.Infof("声明交换器成功 %s", exchangeName)
}

// register 泛型消费端注册函数
func register[T any](exChangeName string, fun func(msg T) error) {
	mq := global.Config.MQ
	cf := global.Config
	// 生成队列名称：交换器名 + 系统UID + 固定后缀，确保节点队列唯一性
	queueName := fmt.Sprintf("%s_%s_queue", exChangeName, cf.System.Uid)

	// 首次初始化时，完成交换器、队列的声明及绑定
	if !mq.InitMQ {
		// 声明业务交换器
		registerExchange(exChangeName)
		// 声明业务队列
		queueDeclare(queueName)
		// 绑定队列到交换器：绑定键为系统UID，确保消息仅路由到当前节点的队列
		err := global.Queue.QueueBind(
			queueName,     // 待绑定的队列名称
			cf.System.Uid, // 绑定键：与生产者的路由键（UID）匹配，实现节点级消息隔离
			exChangeName,  // 绑定的交换器名称
			false,         // 非阻塞：同步绑定，等待结果返回
			nil,           // 额外参数：无特殊配置
		)
		if err != nil {
			logrus.Fatalf("%s 绑定队列失败 %s", queueName, err)
		}
	}

	// 创建队列消费者，关闭自动确认（手动ACK保证消息处理可靠性）
	msgs, err := global.Queue.Consume(
		queueName, // 消费的队列名称
		"",        // 消费者标识：空字符串由MQ自动生成唯一标识
		false,     // 自动确认：关闭，需手动调用Ack确认消息处理完成
		false,     // 非排他性：允许多个消费者消费该队列（当前节点仅一个）
		false,     // 非本地消费者：接收所有发送到队列的消息，包括当前节点生产的
		false,     // 非阻塞：同步创建消费者，等待结果返回
		nil,       // 额外参数：无特殊配置
	)
	if err != nil {
		logrus.Fatalf("%s 创建消费者失败 %s", queueName, err)
	}
	// 消费端创建完成，WaitGroup计数-1
	wg.Done()

	// 循环监听并处理队列消息（通道关闭时退出循环）
	for d := range msgs {
		var data T
		// 解析消息体为指定类型的业务结构体
		err = json.Unmarshal(d.Body, &data)
		if err != nil {
			logrus.Errorf("json解析失败 %s %s", err, string(d.Body))
			d.Ack(false) // 解析失败仍手动ACK，避免消息重复投递
			continue
		}

		// 调用业务处理函数处理消息
		err = fun(data)
		if err == nil {
			d.Ack(false) // 处理成功，手动ACK确认消息（false：仅确认当前消息）
			continue
		}

		// 处理失败时，暂不重新入队（注释Nack逻辑），直接ACK避免死循环
		// d.Nack(false, true) // 拒绝消息，重新入队（可根据业务场景开启）
		d.Ack(false)
	}
	logrus.Errorf("%s 接收队列消息结束", queueName)
}

// sendQueueMessage 发送消息到指定MQ队列
func sendQueueMessage(queueName string, req any) (err error) {
	// 将消息结构体序列化为JSON字节数组，作为MQ消息体
	byteData, _ := json.Marshal(req)
	// 发布消息到MQ：使用默认交换器（空字符串），按队列名路由
	err = global.Queue.Publish(
		"",        // exchange：默认交换器，直接按routing key匹配队列
		queueName, // routing key：目标队列名称，默认交换器会路由到同名队列
		false,     // mandatory：消息无法路由时不返回给生产者
		false,     // immediate：RabbitMQ 3.0+已废弃，固定传false
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型：纯文本（JSON格式）
			Body:        byteData,     // 消息体：JSON序列化后的字节数组
		})
	if err != nil {
		logrus.Errorf("%s 发送消息失败: %v %s", queueName, err, string(byteData))
		return
	}
	logrus.Infof("%s 发送消息成功: %s", queueName, string(byteData))
	return nil
}

// watchHealth 监控MQ连接健康状态并实现自动重连
func watchHealth() {
	// 创建连接关闭通知通道（缓冲大小1，避免MQ推送关闭事件时阻塞）
	closeQueue := make(chan *amqp.Error, 1)
	// 注册连接关闭通知监听：连接断开时，MQ会向该通道发送关闭错误信息
	global.Queue.NotifyClose(closeQueue)

	// 启动独立协程处理重连逻辑，不阻塞主协程
	go func() {
		// 监听关闭通知通道，接收连接关闭的错误信息
		for s := range closeQueue {
			fmt.Println("mq被关闭了", s)
		}
		// 通道关闭，进入重连流程
		fmt.Println("通道被关闭, 等待重连")
		time.Sleep(2 * time.Second) // 重连前延迟，避免频繁重试

		// 循环重连直到成功
		for {
			// 尝试重新初始化MQ连接
			mq, err := core.InitMQ()
			if err != nil {
				logrus.Warnf("mq连接失败, 等待重连 %s", err)
				time.Sleep(2 * time.Second)
				continue
			}
			// 重连成功，更新全局MQ实例
			global.Queue = mq
			// 重启MQ服务，恢复所有消费端和监控
			Run()
			break // 重连成功，退出循环
		}
	}()
}
