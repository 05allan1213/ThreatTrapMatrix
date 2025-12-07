package mq_service

// File: honey_server/service/mq_service/enter.go
// Description: 启动RabbitMQ服务，注册系统所需交换器

import (
	"encoding/json"
	"honey_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Run 注册系统所需的所有RabbitMQ交换器
func Run() {
	cfg := global.Config.MQ
	// 声明创建诱捕IP的交换器
	exchangeDeclare(cfg.CreateIpExchangeName)
	// 声明删除诱捕IP的交换器
	exchangeDeclare(cfg.DeleteIpExchangeName)
	// 声明绑定端口的交换器
	exchangeDeclare(cfg.BindPortExchangeName)
}

// sendExchangeMessage 发送消息到指定的交换器
func sendExchangeMessage(exchangeName, nodeID string, req any) (err error) {
	// 将端口绑定请求参数序列化为JSON字节数据
	byteData, _ := json.Marshal(req)
	// 发布消息到绑定端口的专用交换器
	err = global.Queue.Publish(
		exchangeName, // 目标交换器名称
		nodeID,       // 路由键（节点ID）
		false,        // mandatory：消息无法路由时不强制返回（直接丢弃）
		false,        // immediate：消息无需立即投递（异步处理端口绑定）
		amqp.Publishing{ // 消息内容配置
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 序列化后的端口绑定指令
		})

	// 记录消息发送结果（成功/失败日志）
	if err != nil {
		logrus.Errorf("%s 消息发送失败 %s %s", exchangeName, err, string(byteData))
		return
	}
	logrus.Infof("%s 消息发送成功 %s", exchangeName, string(byteData))
	return
}
