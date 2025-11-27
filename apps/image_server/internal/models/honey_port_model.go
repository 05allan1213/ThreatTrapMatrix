package models

// HoneyPortModel 诱捕端口模型
type HoneyPortModel struct {
	Model
	NodeID    uint   `json:"nodeID"`               // 归属节点ID
	NetID     uint   `json:"netID"`                // 归属网络ID
	HoneyIpID uint   `json:"honeyIpID"`            // 关联诱捕ipID
	ServiceID uint   `json:"serviceID"`            // 服务id
	Port      int    `json:"port"`                 // 服务端口
	DstIP     string `gorm:"size:32" json:"dstIP"` // 目标转发ip
	DstPort   int    `json:"dstPort"`              // 目标转发端口
	Status    int8   `json:"status"`               // 服务状态
}
