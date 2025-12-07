package mq_service

// File: honey_node/service/mq_service/send_update_deploy_status.go
// Description: 实现批量更新部署状态MQ消息的发送功能，上报节点侧诱捕IP端口更新的执行结果（成功/失败）至服务端

import (
	"honey_node/internal/global"
)

// UpdateDeployStatusRequest 更新部署状态上报消息结构体
type UpdateDeployStatusRequest struct {
	NetID    uint       `json:"netID"`    // 子网ID，标识更新操作所属的子网
	IP       string     `json:"ip"`       // 执行更新的诱捕IP地址
	LogID    string     `json:"logID"`    // 日志ID，用于关联更新操作的日志记录
	ErrorMsg string     `json:"errorMsg"` // IP级别的更新错误信息（整体失败时填充）
	PortList []PortInfo `json:"portList"` // 该IP下各端口的更新执行状态列表
}

// PortInfo 端口更新状态信息结构体
type PortInfo struct {
	Port     int    `json:"port"`     // 端口号
	ErrorMsg string `json:"errorMsg"` // 该端口更新失败时的错误信息
}

// SendUpdateDeployStatusMsg 发送更新部署状态MQ消息
func SendUpdateDeployStatusMsg(data UpdateDeployStatusRequest) {
	sendQueueMessage(global.Config.MQ.BatchUpdateDeployStatusTopic, data)
}
