package mq_service

// File: alert_server/service/mq_service/enter.go
// Description: MQ服务启动模块，负责告警队列声明及告警消息消费协程初始化，确保MQ消息接收通道就绪

import (
	"alert_server/internal/global"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Run 初始化MQ服务核心流程：声明告警队列 + 启动告警消息消费协程
func Run() {
	cfg := global.Config.Alert
	// 声明MQ告警队列，配置队列基础属性
	_, err := global.Queue.QueueDeclare(
		cfg.AlertTopic, // 队列名称：从配置读取告警队列主题
		true,           // 持久性：true表示队列数据持久化保存，MQ重启后数据不丢失
		false,          // 自动删除：false表示队列不会自动删除
		false,          // 排他性：false表示非排他队列（允许多消费者连接）
		false,          // 非阻塞：false表示同步声明队列，等待声明完成
		nil,            // 其他额外配置参数：无特殊配置
	)
	if err != nil {
		logrus.Fatalf("声明队列失败: %v", err)
		return
	}

	// 启动告警消息接收协程，异步处理MQ队列中的告警消息（避免阻塞当前启动流程）
	go RevAlertMq()
}

// sendQueueMessage 向指定RabbitMQ队列发送消息
func sendQueueMessage(queueName string, req any) (err error) {
	// 将消息结构体序列化为JSON字节数组，用于MQ消息体传输
	byteData, _ := json.Marshal(req)

	// 调用RabbitMQ Publish方法投递消息
	err = global.Queue.Publish(
		"",        // exchange：使用默认交换机
		queueName, // routing key：指定目标队列名称
		false,     // mandatory：消息无法路由时不返回给生产者
		false,     // immediate：无需立即投递
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
