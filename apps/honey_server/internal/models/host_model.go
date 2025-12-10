package models

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// HostModel 存放主机模型
type HostModel struct {
	Model
	NodeID    uint      `json:"nodeID"`                         // 归属节点ID
	NodeModel NodeModel `gorm:"foreignKey:NodeID" json:"-"`     // 归属节点
	NetID     uint      `gorm:"index:idx_net_id" json:"netID"`  // 归属网络ID
	NetModel  NetModel  `gorm:"foreignKey:NetID" json:"-"`      // 归属网络
	IP        string    `gorm:"size:32;index:idx_ip" json:"ip"` // 主机ip
	Mac       string    `gorm:"size:64" json:"mac"`             // MAC地址
	Manuf     string    `gorm:"size:64" json:"manuf"`           // 厂商信息
}

func (model HostModel) AfterDelete(tx *gorm.DB) error {
	var netModel NetModel
	err := tx.Take(&netModel, model.NetID).Error
	if err != nil {
		logrus.Errorf("网络不存在")
		return err
	}
	tx.Model(&netModel).Update("host_count", gorm.Expr("host_count - 1"))
	return nil
}
