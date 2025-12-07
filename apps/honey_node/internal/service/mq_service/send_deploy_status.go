package mq_service

// File: honey_node/service/mq_service/send_deploy_status.go
// Description: 实现部署状态消息的MQ发送功能，用于上报诱捕IP部署的状态、进度、错误信息等数据到指定MQ队列

import (
	"honey_node/internal/global"
)

// DeployStatusRequest 部署状态上报请求结构体
type DeployStatusRequest struct {
	NetID    uint   `json:"netID"`    // 子网ID，标识部署所属的子网
	IP       string `json:"ip"`       // 部署的诱捕IP地址
	Mac      string `json:"mac"`      // IP绑定的MAC地址
	LinkName string `json:"linkName"` // 网络接口名称
	LogID    string `json:"logID"`    // 日志ID，用于关联部署操作的日志记录
	ErrorMsg string `json:"errorMsg"` // 部署失败时的错误信息，成功时为空
	Manuf    string `json:"manuf"`    // 厂商信息
}

// SendDeployStatusMsg 发送部署状态消息到MQ队列
func SendDeployStatusMsg(data DeployStatusRequest) {
	sendQueueMessage(global.Config.MQ.BatchDeployStatusTopic, data)
}
