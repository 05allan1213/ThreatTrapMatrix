package mq_service

// File: matrix_server/service/mq_service/send_batch_remove_deploy_msg.go
// Description: MQ消息服务模块，实现批量删除部署消息的组装与发送功能，基于RabbitMQ完成消息下发至指定节点

import (
	"matrix_server/internal/global"
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
	return sendExchangeMessage(global.Config.MQ.BatchRemoveDeployExchangeName, nodeUID, req)
}
