package models

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"errors"
)

// NodeNetworkModel 节点网卡模型
type NodeNetworkModel struct {
	Model
	NodeID    uint      `json:"nodeID"`                     // 归属节点ID
	NodeModel NodeModel `gorm:"foreignKey:NodeID" json:"-"` // 归属节点
	Network   string    `gorm:"32" json:"network"`          // 网卡名称
	IP        string    `gorm:"32" json:"ip"`               // 探针ip
	Mask      int8      `json:"mask"`                       // 子网掩码 8-32
	Gateway   string    `gorm:"32" json:"gateway"`          // 网关
	Status    int8      `json:"status"`                     // 网卡启用状态 1 启用 2 未启用
}

func (n *NodeNetworkModel) BeforeDelete(tx *gorm.DB) error {
	// 检查网卡是否启用，如果未启用则直接返回
	if n.Status == 2 {
		return nil
	}

	// 根据节点ID和网络名称查询对应的网络模型
	var net NetModel
	err := tx.Take(&net, "node_id = ? and network = ?", n.NodeID, n.Network).Error
	if err != nil {
		// 如果找不到对应网络，说明该网卡未启用，直接返回
		return nil
	}

	// 查询该网络下是否有诱捕IP
	var count int64
	tx.Model(HoneyIpModel{}).Where("net_id = ?", net.ID).Count(&count)
	if count > 0 {
		// 如果存在诱捕IP，则不允许删除该网卡
		return errors.New("此网卡的网络存在诱捕ip，不可删除")
	}

	// 删除与该网络关联的所有主机记录
	var hostList []HostModel
	tx.Find(&hostList, "net_id = ?", net.ID).Delete(&hostList)

	// 删除网络记录
	tx.Delete(&net)

	// 记录日志信息
	logrus.Infof("关联删除主机记录 %d", len(hostList))
	logrus.Infof("关联删除网络 %s", net.Title)

	return nil
}
