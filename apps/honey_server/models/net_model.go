package models

import "gorm.io/gorm"

// NetModel 网络模型
type NetModel struct {
	gorm.Model
	NodeID             uint      `json:"nodeID"`                        // 归属节点ID
	NodeModel          NodeModel `gorm:"foreignKey:NodeID" json:"-"`    // 归属节点
	Title              string    `gorm:"32" json:"title"`               // 网络名称
	Network            string    `gorm:"32" json:"network"`             // 网卡名称
	IP                 string    `gorm:"32" json:"ip"`                  // 探针ip
	Mask               int8      `json:"mask"`                          // 子网掩码 8-32
	Gateway            string    `gorm:"32" json:"gateway"`             // 网关
	HostCount          int       `json:"hostCount"`                     // 存放资产（子网中活跃的主机）
	HoneyIpCount       int       `json:"honeyIpCount"`                  // 诱捕ip数
	ScanStatus         int8      `json:"scanStatus"`                    // 扫描状态
	ScanProgress       float64   `json:"scanProgress"`                  // 扫描进度
	CanUseHoneyIPRange string    `gorm:"256" json:"canUseHoneyIPRange"` // 能够使用的诱捕ip范围
}
