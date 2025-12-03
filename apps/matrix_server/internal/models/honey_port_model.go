package models

// HoneyPortModel 诱捕端口模型
type HoneyPortModel struct {
	Model
	NodeID       uint         `json:"nodeID"`                                 // 归属节点ID
	NodeModel    NodeModel    `gorm:"foreignKey:NodeID" json:"-"`             // 归属节点
	NetID        uint         `json:"netID"`                                  // 归属网络ID
	NetModel     NetModel     `gorm:"foreignKey:NetID" json:"-"`              // 归属网络
	HoneyIpID    uint         `gorm:"index:idx_honey_ip_id" json:"honeyIpID"` // 关联诱捕ipID
	HoneyIpModel HoneyIpModel `gorm:"foreignKey:HoneyIpID" json:"-"`          // 关联诱捕ip
	ServiceID    uint         `json:"serviceID"`                              // 服务id
	ServiceModel ServiceModel `gorm:"foreignKey:ServiceID" json:"-"`          // 关联服务
	IP           string       `gorm:"size:32;index:idx_ip" json:"ip"`         // 服务ip
	Port         int          `json:"port"`                                   // 服务端口
	DstIP        string       `gorm:"size:32" json:"dstIP"`                   // 目标转发ip
	DstPort      int          `json:"dstPort"`                                // 目标转发端口
	Status       int8         `json:"status"`                                 // 服务状态
}
