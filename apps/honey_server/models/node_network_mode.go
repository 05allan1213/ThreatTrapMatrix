package models

// NodeNetworkModel 节点网卡模型
type NodeNetworkModel struct {
	Model
	NodeID    uint      `json:"nodeID"`                     // 归属节点ID
	NodeModel NodeModel `gorm:"foreignKey:NodeID" json:"-"` // 归属节点
	Network   string    `gorm:"32" json:"network"`          // 网卡名称
	IP        string    `gorm:"32" json:"ip"`               // 探针ip
	Mask      int8      `json:"mask"`                       // 子网掩码 8-32
	Gateway   string    `gorm:"32" json:"gateway"`          // 网关
	Status    int8      `json:"status"`                     // 网卡启用状态
}
