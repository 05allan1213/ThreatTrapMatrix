package mq_service

// File: honey_server/service/mq_service/send_bind_port_msg.go
// Description: 绑定诱捕端口消息发送服务，负责将端口绑定请求封装为消息并发送到RabbitMQ交换器

import (
	"encoding/json"
	"honey_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// BindPortRequest 绑定诱捕端口的消息结构体
type BindPortRequest struct {
	IP       string     `json:"ip"`       // 要绑定端口的诱捕IP地址
	PortList []PortInfo `json:"portList"` // 端口绑定配置列表（外部端口与目标服务的映射）
	LogID    string     `json:"logID"`    // 操作日志ID（用于追踪端口绑定操作的链路）
}

// PortInfo 单个端口的绑定信息结构体
type PortInfo struct {
	IP       string `json:"ip"`       // 关联的诱捕IP地址
	Port     int    `json:"port"`     // 对外暴露的诱捕端口号
	DestIP   string `json:"destIP"`   // 目标转发服务的IP地址
	DestPort int    `json:"destPort"` // 目标转发服务的端口号
}

// SendBindPortMsg 发送端口绑定请求的消息到RabbitMQ
func SendBindPortMsg(nodeUID string, req BindPortRequest) {
	// 将端口绑定请求参数序列化为JSON字节数据
	byteData, _ := json.Marshal(req)
	cfg := global.Config.MQ // 获取MQ全局配置

	// 发布消息到绑定端口的专用交换器
	err := global.Queue.Publish(
		cfg.BindPortExchangeName, // 目标交换器名称（端口绑定专用）
		nodeUID,                  // 路由键（指定消息发送到的目标节点）
		false,                    // mandatory：消息无法路由时不强制返回（直接丢弃）
		false,                    // immediate：消息无需立即投递（异步处理端口绑定）
		amqp.Publishing{ // 消息内容配置
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 序列化后的端口绑定指令
		})

	// 记录消息发送结果（成功/失败日志）
	if err != nil {
		logrus.Errorf("端口绑定消息发送失败 %s %s", err, string(byteData))
	} else {
		logrus.Infof("端口绑定消息发送成功 %s ", string(byteData))
	}
}
