package mq_service

// File: honey_node/service/mq_service/send_remove_deploy_status.go
// Description: MQ消息服务模块，实现删除部署状态消息的组装与发送功能，用于将单个IP的删除部署执行状态通过RabbitMQ反馈至指定队列

import (
	"honey_node/internal/global"
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
	sendQueueMessage(global.Config.MQ.BatchRemoveDeployStatusTopic, data)
}
