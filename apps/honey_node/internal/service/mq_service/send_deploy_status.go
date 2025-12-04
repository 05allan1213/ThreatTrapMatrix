package mq_service

// File: honey_node/service/mq_service/send_deploy_status.go
// Description: 实现部署状态消息的MQ发送功能，用于上报诱捕IP部署的状态、进度、错误信息等数据到指定MQ队列

import (
	"encoding/json"
	"honey_node/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// DeployStatusRequest 部署状态上报请求结构体
type DeployStatusRequest struct {
	NetID    uint    `json:"netID"`    // 子网ID，标识部署所属的子网
	IP       string  `json:"ip"`       // 部署的诱捕IP地址
	Mac      string  `json:"mac"`      // IP绑定的MAC地址
	LinkName string  `json:"linkName"` // 网络接口名称
	LogID    string  `json:"logID"`    // 日志ID，用于关联部署操作的日志记录
	ErrorMsg string  `json:"errorMsg"` // 部署失败时的错误信息，成功时为空
	Progress float64 `json:"progress"` // 部署进度（1-100的小数）
}

// SendDeployStatusMsg 发送部署状态消息到MQ队列
func SendDeployStatusMsg(data DeployStatusRequest) {
	// 将部署状态数据序列化为JSON字节数组
	byteData, _ := json.Marshal(data)
	// 获取全局配置中的MQ配置信息
	cfg := global.Config.MQ

	// 发布消息到MQ队列
	err := global.Queue.Publish(
		"",                         // exchange：使用默认交换机
		cfg.BatchDeployStatusTopic, // routing key：指定告警主题作为路由键
		false,                      // mandatory：不强制要求消息必须路由到队列
		false,                      // immediate：不要求立即投递
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 消息体：序列化后的部署状态数据
		})

	// 处理消息发送失败的情况
	if err != nil {
		logrus.Errorf("发送失败: %v %s", err, string(byteData))
		return
	}

	// 记录消息发送成功日志
	logrus.Infof("发送成功: %s", string(byteData))
}
