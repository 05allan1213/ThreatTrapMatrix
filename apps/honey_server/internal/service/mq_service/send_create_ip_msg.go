package mq_service

// File: honey_server/service/mq_service/send_create_ip_msg.go
// Description: 创建诱捕IP消息发送服务，负责将创建IP的请求封装为消息并发送到RabbitMQ交换器

import (
	"honey_server/internal/global"
)

// CreateIPRequest 创建诱捕IP的消息结构体
type CreateIPRequest struct {
	HoneyIPID uint   `json:"honeyIpID"` // 诱捕ipID
	IP        string `json:"ip"`        // 要创建的诱捕IP地址
	Mask      int8   `json:"mask"`      // 子网掩码位数（如24）
	Network   string `json:"network"`   // 绑定的物理网卡名称
	LogID     string `json:"logID"`     // 操作日志ID（用于追踪操作链路）
	IsTan     bool   `json:"isTan"`     // 是否是探针ip
}

// SendCreateIPMsg 发送创建诱捕IP的消息到RabbitMQ
func SendCreateIPMsg(nodeUID string, req CreateIPRequest) error {
	return sendExchangeMessage(global.Config.MQ.CreateIpExchangeName, nodeUID, req)
}
