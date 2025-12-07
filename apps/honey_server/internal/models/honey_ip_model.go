package models

// HoneyIpModel 诱捕ip模型
type HoneyIpModel struct {
	Model
	NodeID         uint             `json:"nodeID"`                         // 归属节点ID
	NodeModel      NodeModel        `gorm:"foreignKey:NodeID" json:"-"`     // 归属节点
	NetID          uint             `gorm:"index:idx_net_id" json:"netID"`  // 归属网络ID
	NetModel       NetModel         `gorm:"foreignKey:NetID" json:"-"`      // 归属网络
	PortList       []HoneyPortModel `gorm:"foreignKey:HoneyIpID" json:"-"`  // 诱捕ip的端口列表
	IP             string           `gorm:"size:32;index:idx_ip" json:"ip"` // 诱捕ip
	Mac            string           `gorm:"size:64" json:"mac"`             // MAC地址
	Network        string           `gorm:"size:32" json:"network"`         // 网卡名称
	Status         int8             `json:"status"`                         // 部署状态 1 创建中 2 运行中 3 失败 4 删除中
	ErrorMsg       string           `gorm:"size:64" json:"errorMsg"`        // 错误信息
	HostTemplateID uint             `json:"hostTemplateID"`                 // 所属主机模板
}
