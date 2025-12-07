package mq_service

// File: honey_server/service/mq_service/send_delete_ip_msg.go
// Description: 删除诱捕IP消息发送服务，负责将批量删除IP的请求封装为消息并发送到RabbitMQ交换器

import (
	"honey_server/internal/global"
)

// DeleteIPRequest 批量删除诱捕IP的消息结构体
type DeleteIPRequest struct {
	IpList []IpInfo `json:"ipList"` // 待删除的IP信息列表
	NetID  uint     `json:"netID"`  // 网络ID
	LogID  string   `json:"logID"`  // 操作日志ID（用于追踪批量删除操作的链路）
}

// IpInfo 单个IP的删除信息结构体
type IpInfo struct {
	HoneyIPID uint   `json:"honeyIpID"` // 诱捕ipID（关联数据库主键）
	IP        string `json:"ip"`        // 要删除的诱捕IP地址
	Network   string `json:"network"`   // 该IP绑定的物理网卡名称
	IsTan     bool   `json:"isTan"`     // 是否是探针ip
}

// SendDeleteIPMsg 发送批量删除诱捕IP的消息到RabbitMQ
func SendDeleteIPMsg(nodeUID string, req DeleteIPRequest) error {
	return sendExchangeMessage(global.Config.MQ.DeleteIpExchangeName, nodeUID, req)
}
