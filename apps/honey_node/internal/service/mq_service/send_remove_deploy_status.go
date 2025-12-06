package mq_service

// File: honey_node/service/mq_service/send_remove_deploy_status.go
// Description: MQ消息服务模块，实现删除部署状态消息的组装与发送功能，用于将单个IP的删除部署执行状态通过RabbitMQ反馈至指定队列

import (
	"encoding/json"
	"honey_node/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// RemoveDeployStatusRequest 删除部署状态的MQ消息请求结构体
type RemoveDeployStatusRequest struct {
	NetID    uint   `json:"netID"`    // 子网ID，关联目标操作子网
	IP       string `json:"ip"`       // 执行删除部署操作的IP地址
	LogID    string `json:"logID"`    // 日志ID，用于关联操作全链路日志
	ErrorMsg string `json:"errorMsg"` // 删除部署执行失败时的错误信息，成功则为空
}

// SendRemoveDeployStatusMsg 发送单个IP的删除部署状态MQ消息
func SendRemoveDeployStatusMsg(data RemoveDeployStatusRequest) {
	// 将删除部署状态结构体序列化为JSON字节数据
	byteData, _ := json.Marshal(data)
	// 获取全局MQ配置信息
	cfg := global.Config.MQ

	// 向RabbitMQ发布删除部署状态消息
	err := global.Queue.Publish(
		"",                               // 交换机名称（空表示使用默认交换机）
		cfg.BatchRemoveDeployStatusTopic, // 路由键（删除部署状态反馈队列标识）
		false,                            // mandatory：是否强制要求消息路由到队列
		false,                            // immediate：是否要求立即投递消息
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 消息体（JSON序列化后的状态数据）
		})
	// 消息发送失败时记录错误日志（含错误信息与消息内容）
	if err != nil {
		logrus.Errorf("发送失败: %v %s", err, string(byteData))
		return
	}
	// 消息发送成功时记录日志
	logrus.Infof("发送成功: %s", string(byteData))
}
