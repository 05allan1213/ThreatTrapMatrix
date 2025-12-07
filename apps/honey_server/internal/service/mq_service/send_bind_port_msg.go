package mq_service

import (
	"honey_server/internal/global"
)

// File: honey_server/service/mq_service/send_bind_port_msg.go
// Description: 绑定诱捕端口消息发送服务，负责将端口绑定请求封装为消息并发送到RabbitMQ交换器

// BindPortRequest 绑定诱捕端口的消息结构体
type BindPortRequest struct {
	IP        string     `json:"ip"`        // 要绑定端口的诱捕IP地址
	PortList  []PortInfo `json:"portList"`  // 端口绑定配置列表（外部端口与目标服务的映射）
	HoneyIpID uint       `json:"honeyIpID"` // 绑定的诱捕IPID
	LogID     string     `json:"logID"`     // 操作日志ID（用于追踪端口绑定操作的链路）
}

// PortInfo 单个端口的绑定信息结构体
type PortInfo struct {
	IP       string `json:"ip"`       // 关联的诱捕IP地址
	Port     int    `json:"port"`     // 对外暴露的诱捕端口号
	DestIP   string `json:"destIP"`   // 目标转发服务的IP地址
	DestPort int    `json:"destPort"` // 目标转发服务的端口号
}

// SendBindPortMsg 发送绑定端口的消息到RabbitMQ
func SendBindPortMsg(nodeUID string, req BindPortRequest) error {
	return sendExchangeMessage(global.Config.MQ.BindPortExchangeName, nodeUID, req)
}
