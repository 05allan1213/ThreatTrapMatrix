package mq_service

// File: matrix_server/service/mq_service/send_batch_deploy_msg.go
// Description: 实现批量部署指令的MQ发送功能，用于将子网诱捕IP批量部署指令下发至指定节点的MQ队列

import (
	"matrix_server/internal/global"
)

// BatchDeployRequest 批量部署指令发送请求结构体
type BatchDeployRequest struct {
	NetID   uint       `json:"netID"`   // 子网ID，标识部署的目标子网
	LogID   string     `json:"logID"`   // 日志ID，用于关联部署操作的日志记录
	Network string     `json:"network"` // 网卡名称，指定IP所属的网络环境
	TanIp   string     `json:"tanIp"`   // 探针IP
	IPList  []DeployIp `json:"ipList"`  // 待部署IP列表，包含每个IP的具体配置
}

// DeployIp 单IP部署配置结构体
type DeployIp struct {
	Ip       string     `json:"ip"`       // 待部署的诱捕IP地址
	Mask     int8       `json:"mask"`     // IP子网掩码
	PortList []PortInfo `json:"portList"` // 该IP关联的端口转发配置列表
}

// PortInfo 端口转发配置结构体
type PortInfo struct {
	IP       string `json:"ip"`       // 源IP地址
	Port     int    `json:"port"`     // 源端口号
	DestIP   string `json:"destIP"`   // 目标IP地址
	DestPort int    `json:"destPort"` // 目标端口号
}

// SendBatchDeployMsg 发送批量部署指令到MQ队列
func SendBatchDeployMsg(nodeUID string, req BatchDeployRequest) (err error) {
	return SendExchangeMessage(global.Config.MQ.BatchDeployExchangeName, nodeUID, req)
}
