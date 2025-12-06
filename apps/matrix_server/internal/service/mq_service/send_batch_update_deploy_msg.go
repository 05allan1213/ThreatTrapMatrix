package mq_service

// File: matrix_server/service/mq_service/send_batch_update_deploy_msg.go
// Description: 实现批量更新部署指令的MQ发送功能，用于将子网诱捕IP的部署配置更新指令下发至指定节点的MQ队列

import (
	"matrix_server/internal/global"
)

// BatchUpdateDeployRequest 批量更新部署指令发送请求结构体
type BatchUpdateDeployRequest struct {
	NetID    uint       `json:"netID"`    // 子网ID，标识更新操作所属的子网
	LogID    string     `json:"logID"`    // 日志ID，用于关联更新操作的日志记录
	IpList   []string   `json:"ipList"`   // 待更新配置的诱捕IP列表（主机模板变更的IP）
	PortList []PortInfo `json:"portList"` // 新的端口转发配置列表（基于新主机模板）
}

// SendBatchUpdateDeployMsg 发送批量更新部署指令到MQ队列
func SendBatchUpdateDeployMsg(nodeUID string, req BatchUpdateDeployRequest) (err error) {
	return sendExchangeMessage(global.Config.MQ.BatchUpdateDeployExchangeName, nodeUID, req)
}
