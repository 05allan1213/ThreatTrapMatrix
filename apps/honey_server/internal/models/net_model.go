package models

import (
	"errors"
	"fmt"
	"honey_server/internal/utils/ip"
	"net"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NetModel 网络模型
type NetModel struct {
	Model
	NodeID             uint      `json:"nodeID"`                             // 归属节点ID
	NodeModel          NodeModel `gorm:"foreignKey:NodeID" json:"-"`         // 归属节点
	Title              string    `gorm:"size:32" json:"title"`               // 网络名称
	Network            string    `gorm:"size:32" json:"network"`             // 网卡名称
	IP                 string    `gorm:"size:32" json:"ip"`                  // 探针ip
	Mask               int8      `json:"mask"`                               // 子网掩码 8-32
	Gateway            string    `gorm:"size:32" json:"gateway"`             // 网关
	HostCount          int       `json:"hostCount"`                          // 存放资产（子网中活跃的主机）
	HoneyIpCount       int       `json:"honeyIpCount"`                       // 诱捕ip数
	ScanStatus         int8      `json:"scanStatus"`                         // 扫描状态  0 待扫描  1 扫描完成  2 扫描中
	ScanProgress       float64   `json:"scanProgress"`                       // 扫描进度
	CanUseHoneyIPRange string    `gorm:"size:256" json:"canUseHoneyIPRange"` // 能够使用的诱捕ip范围
}

// Subnet 返回网络模型的子网信息
func (model NetModel) Subnet() string {
	return fmt.Sprintf("%s/%d", model.IP, model.Mask)
}

// InSubnet 判断给定的IP地址是否属于当前网络模型的子网
func (model NetModel) InSubnet(ip string) bool {
	_, _net, _ := net.ParseCIDR(model.Subnet())
	return _net.Contains(net.ParseIP(ip))
}

// IpRange 获取网络模型中的IP范围
func (model NetModel) IpRange() (ipRange []string, err error) {
	return ip.ParseIPRange(model.CanUseHoneyIPRange)
}

func (model NetModel) BeforeDelete(tx *gorm.DB) error {
	// 校验当前网络下是否存在诱捕IP，存在则禁止删除
	var count int64
	tx.Model(&HoneyIpModel{}).Where("net_id = ?", model.ID).Count(&count)
	if count > 0 {
		return errors.New("存在诱捕ip，不能删除网络")
	}

	// 查询关联的节点网卡记录
	var nodeNet NodeNetworkModel
	err := tx.Take(&nodeNet, "node_id = ? and network = ?", model.NodeID, model.Network).Error
	if err != nil {
		// 无关联网卡记录则直接返回，允许删除
		return nil
	}

	// 删除关联的节点网卡下的所有主机记录
	var hostList []HostModel
	tx.Find(&hostList, "net_id = ?", model.ID).Delete(&hostList)
	logrus.Infof("关联删除主机 %d个", len(hostList))

	// 将关联网卡状态重置为未启用（状态2）
	tx.Model(&nodeNet).Update("status", 2)
	return nil
}
