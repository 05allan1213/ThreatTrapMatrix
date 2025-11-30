package models

// HoneyIpModel 诱捕ip模型
type HoneyIpModel struct {
	Model
	NodeID    uint      `json:"nodeID"`                     // 归属节点ID
	NodeModel NodeModel `gorm:"foreignKey:NodeID" json:"-"` // 归属节点
	NetID     uint      `json:"netID"`                      // 归属网络ID
	NetModel  NetModel  `gorm:"foreignKey:NetID" json:"-"`  // 归属网络
	IP        string    `gorm:"32" json:"ip"`               // 诱捕ip
	Mac       string    `gorm:"64" json:"mac"`              // MAC地址
	Network   string    `gorm:"32" json:"network"`          // 网卡名称
	Status    int8      `json:"status"`                     // 部署状态 1 创建中 2 运行中 3 失败 4 删除中
}
