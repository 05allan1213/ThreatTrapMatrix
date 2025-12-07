package mq_service

// File: honey_node/service/mq_service/send_alert.go
// Description: 消息队列告警发送模块，负责将告警数据封装为标准MQ消息，发送至指定告警队列

import (
	"honey_node/internal/global"
)

// AlertMsgType 告警MQ消息结构体，定义告警数据的标准传输格式
type AlertMsgType struct {
	NodeUid   string `json:"nodeUid"`   // 节点唯一标识
	SrcIp     string `json:"srcIp"`     // 攻击源IP地址
	SrcPort   int    `json:"srcPort"`   // 攻击源端口
	DestIp    string `json:"destIp"`    // 攻击目标IP地址
	DestPort  int    `json:"destPort"`  // 攻击目标端口
	Timestamp string `json:"timestamp"` // 告警发生时间
	Signature string `json:"signature"` // 告警规则描述
	Level     int8   `json:"level"`     // 告警级别
	Body      string `json:"body"`      // HTTP响应体内容（仅HTTP相关告警有效)
	Payload   string `json:"payload"`   // 告警关联数据包负载内容
}

// SendAlertMsg 将告警数据序列化为JSON格式，发送至MQ指定告警队列
func SendAlertMsg(data AlertMsgType) {
	sendQueueMessage(global.Config.MQ.AlertTopic, data)
}
