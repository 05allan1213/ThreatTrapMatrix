package mq_service

// File: honey_node/service/mq_service/send_alert.go
// Description: 消息队列告警发送模块，负责将告警数据封装为标准MQ消息，发送至指定告警队列

import (
	"encoding/json"
	"honey_node/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// AlertMsgType 告警MQ消息结构体，定义告警数据的标准传输格式
type AlertMsgType struct {
	NodeUid          string `json:"nodeUid"`          // 节点唯一标识
	SrcIp            string `json:"srcIp"`            // 攻击源IP地址
	SrcPort          int    `json:"srcPort"`          // 攻击源端口
	DestIp           string `json:"destIp"`           // 攻击目标IP地址
	DestPort         int    `json:"destPort"`         // 攻击目标端口
	Timestamp        string `json:"timestamp"`        // 告警发生时间
	Signature        string `json:"signature"`        // 告警规则描述
	Level            int8   `json:"level"`            // 告警级别
	HttpResponseBody string `json:"httpResponseBody"` // HTTP响应体内容（仅HTTP相关告警有效）
	Payload          string `json:"payload"`          // 告警关联数据包负载内容
}

// SendAlertMsg 将告警数据序列化为JSON格式，发送至MQ指定告警队列
func SendAlertMsg(data AlertMsgType) {
	// 将告警结构体序列化为JSON字节流
	byteData, _ := json.Marshal(data)
	cfg := global.Config.MQ

	// 向MQ发送告警消息，使用配置中指定的告警队列路由键
	err := global.Queue.Publish(
		"",             // exchange：空字符串表示使用默认交换机
		cfg.AlertTopic, // routing key：告警队列主题（从配置读取）
		false,          // mandatory：消息无法路由时是否返回生产者（此处关闭）
		false,          // immediate：队列无消费者时是否立即返回（此处关闭）
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型：纯文本（实际为JSON格式）
			Body:        byteData,     // 消息体：JSON序列化后的告警数据
		})
	if err != nil {
		logrus.Errorf("发送告警信息失败: %v %s", err, string(byteData))
		return
	}
	logrus.Infof("发送告警成功: %s", string(byteData))
}
