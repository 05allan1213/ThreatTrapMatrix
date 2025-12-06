package models

import "fmt"

// TaskModel 任务模型
type TaskModel struct {
	Model
	TaskID          string              `json:"taskID"`                       // 任务ID
	Type            int8                `json:"type"`                         // 任务类型 1 批量部署
	BatchDeployData *BatchDeployRequest `gorm:"serializer:json" json:"value"` // 批量部署参数 值 json字符串
	Status          int8                `json:"status"`                       // 任务状态 0 运行中 1 运行完成
}

// BatchDeployRequest MQ消费的批量部署请求结构体
type BatchDeployRequest struct {
	NetID   uint       `json:"netID"`   // 子网ID
	LogID   string     `json:"logID"`   // 日志ID
	Network string     `json:"network"` // 网卡名称
	IPList  []DeployIp `json:"ipList"`  // 待部署IP列表
}

// DeployIp 单IP部署配置信息结构体
type DeployIp struct {
	Ip       string     `json:"ip"`       // 待部署的诱捕IP地址
	IsTan    bool       `json:"isTan"`    // 是否为探针IP标识
	Mask     int8       `json:"mask"`     // IP子网掩码
	PortList []PortInfo `json:"portList"` // 该IP关联的端口转发配置列表
}

// PortInfo 端口信息结构体
type PortInfo struct {
	IP       string `json:"ip"`       // 源ip
	Port     int    `json:"port"`     // 源端口
	DestIP   string `json:"destIP"`   // 目标ip
	DestPort int    `json:"destPort"` // 目标端口
}

// LocalAddr 本地监听地址
func (p PortInfo) LocalAddr() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

// TargetAddr 目标服务地址
func (p PortInfo) TargetAddr() string {
	return fmt.Sprintf("%s:%d", p.DestIP, p.DestPort)
}
