package mq_service

// File: honey_server/service/mq_service/send_msg.go
// Description: 创建诱捕IP消息发送服务，负责将创建IP的请求封装为消息并发送到RabbitMQ交换器

import (
	"encoding/json"
	"honey_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// CreateIPRequest 创建诱捕IP的消息结构体
type CreateIPRequest struct {
	HoneyIPID uint   `json:"honeyIpID"` // 诱捕ipID
	IP        string `json:"ip"`        // 要创建的诱捕IP地址
	Mask      int8   `json:"mask"`      // 子网掩码位数（如24）
	Network   string `json:"network"`   // 绑定的物理网卡名称
	LogID     string `json:"logID"`     // 操作日志ID（用于追踪操作链路）
}

// SendCreateIPMsg 发送创建诱捕IP的消息到RabbitMQ
func SendCreateIPMsg(nodeUID string, req CreateIPRequest) {
	// 将请求参数序列化为JSON字节数据
	byteData, _ := json.Marshal(req)
	cfg := global.Config.MQ // 获取MQ全局配置

	// 发布消息到创建IP的交换器
	err := global.Queue.Publish(
		cfg.CreateIpExchangeName, // 目标交换器名称（创建IP专用）
		nodeUID,                  // 路由键（指定消息发送到的节点）
		false,                    // mandatory：消息无法路由时是否返回（false直接丢弃）
		false,                    // immediate：消息无法立即投递时是否返回（false异步处理）
		amqp.Publishing{ // 消息内容配置
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 序列化后的消息体
		})

	// 记录消息发送结果
	if err != nil {
		logrus.Errorf("消息发送失败 %s %s", err, string(byteData))
	} else {
		logrus.Infof("消息发送成功 %s ", string(byteData))
	}
}
