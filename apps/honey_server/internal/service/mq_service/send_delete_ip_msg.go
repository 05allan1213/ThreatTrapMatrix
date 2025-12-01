package mq_service

// File: honey_server/service/mq_service/send_delete_ip_msg.go
// Description: 删除诱捕IP消息发送服务，负责将批量删除IP的请求封装为消息并发送到RabbitMQ交换器

import (
	"encoding/json"
	"honey_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// DeleteIPRequest 批量删除诱捕IP的消息结构体
type DeleteIPRequest struct {
	IpList []IpInfo `json:"ipList"` // 待删除的IP信息列表
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
func SendDeleteIPMsg(nodeUID string, req DeleteIPRequest) {
	// 将批量删除请求参数序列化为JSON字节数据
	byteData, _ := json.Marshal(req)
	cfg := global.Config.MQ // 获取MQ全局配置

	// 发布消息到删除IP的专用交换器
	err := global.Queue.Publish(
		cfg.DeleteIpExchangeName, // 目标交换器名称（删除IP专用）
		nodeUID,                  // 路由键（指定消息发送到的目标节点）
		false,                    // mandatory：消息无法路由时不强制返回（直接丢弃）
		false,                    // immediate：消息无需立即投递（异步处理批量删除）
		amqp.Publishing{ // 消息内容配置
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 序列化后的批量删除指令
		})

	// 记录消息发送结果（成功/失败日志）
	if err != nil {
		logrus.Errorf("批量删除IP消息发送失败 %s %s", err, string(byteData))
	} else {
		logrus.Infof("批量删除IP消息发送成功 %s ", string(byteData))
	}
}
