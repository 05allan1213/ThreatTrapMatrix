package models

import "gorm.io/gorm"

// HostModel 存放主机模型
type HostModel struct {
	gorm.Model
	NodeID    uint      `json:"nodeID"`                     // 归属节点ID
	NodeModel NodeModel `gorm:"foreignKey:NodeID" json:"-"` // 归属节点
	NetID     uint      `json:"netID"`                      // 归属网络ID
	NetModel  NetModel  `gorm:"foreignKey:NetID" json:"-"`  // 归属网络
	IP        string    `gorm:"32" json:"ip"`               // 主机ip
	Mac       string    `gorm:"64" json:"mac"`              // MAC地址
	Manuf     string    `gorm:"64" json:"manuf"`            // 厂商信息
}
