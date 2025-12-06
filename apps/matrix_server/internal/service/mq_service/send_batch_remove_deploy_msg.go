package mq_service

// File: matrix_server/service/mq_service/send_batch_remove_deploy_msg.go
// Description: MQ消息服务模块，实现批量删除部署消息的组装与发送功能，基于RabbitMQ完成消息下发至指定节点

import (
	"encoding/json"
	"matrix_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// BatchRemoveDeployRequest 批量删除部署的MQ消息请求结构体
type BatchRemoveDeployRequest struct {
	NetID  uint             `json:"netID"`  // 子网ID，关联目标操作子网
	LogID  string           `json:"logID"`  // 日志ID，用于关联操作全链路日志
	TanIp  string           `json:"tanIp"`  // 探针IP
	IPList []RemoveDeployIp `json:"ipList"` // 待删除部署的IP信息列表
}

// RemoveDeployIp 批量删除部署请求中的IP信息结构体
type RemoveDeployIp struct {
	Ip       string `json:"ip"`       // 待删除部署的IP地址
	LinkName string `json:"linkName"` // IP对应的网络链路名称
}

// SendBatchRemoveDeployMsg 发送批量删除部署的MQ消息
func SendBatchRemoveDeployMsg(nodeUID string, req BatchRemoveDeployRequest) (err error) {
	// 将请求结构体序列化为JSON字节数据
	byteData, _ := json.Marshal(req)
	// 获取全局MQ配置信息
	cfg := global.Config.MQ
	// 向RabbitMQ发布批量删除部署消息
	err = global.Queue.Publish(
		cfg.BatchRemoveDeployExchangeName, // 消息交换机名称
		nodeUID,                           // 路由键（节点UID）
		false,                             // mandatory：是否强制要求消息路由到队列
		false,                             // immediate：是否要求立即投递消息
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 消息体（JSON序列化后的请求数据）
		})
	// 消息发送失败时记录错误日志（含错误信息与消息内容）
	if err != nil {
		logrus.Errorf("消息发送失败 %s %s", err, string(byteData))
		return err
	}
	// 消息发送成功时记录日志
	logrus.Infof("消息发送成功 %s", string(byteData))
	return
}
