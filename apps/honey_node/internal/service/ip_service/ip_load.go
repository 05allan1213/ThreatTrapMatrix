package ip_service

// File: honey_node/service/ip_service/ip_load.go
// Description: IP服务模块，负责应用启动时加载数据库IP配置，校验网卡与IP一致性并初始化网络配置

import (
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/utils"
	"honey_node/internal/utils/info"

	"github.com/sirupsen/logrus"
)

// IPLoad 应用启动时加载数据库中的IP配置记录，校验并初始化网卡与IP地址
func IPLoad() {
	// 从数据库查询所有IP配置记录
	var ipList []models.IpModel
	global.DB.Find(&ipList)

	// 获取本机所有网卡及对应IP信息，用于后续校验
	networkMap, err := info.GetNetworkInterfaces()
	if err != nil {
		logrus.Fatalf("获取网卡错误 %s", err)
		return
	}

	// 遍历每条IP配置记录，执行校验或创建逻辑
	for _, model := range ipList {
		// 判断配置的网卡是否已存在于本机
		ips, ok := networkMap[model.LinkName]
		if ok {
			// 网卡存在，校验配置的IP是否在该网卡的IP列表中
			if !utils.InList(ips, model.Ip) {
				logrus.Errorf("网卡 %s 对应的ip地址错误 %v %s", model.LinkName, ips, model.Ip)
				continue // IP不匹配，跳过该配置
			}
			continue // 网卡和IP均匹配，无需额外操作
		}

		// 网卡不存在，创建网卡并配置IP
		_, err := SetIp(SetIpRequest{
			Ip:       model.Ip,
			Mask:     model.Mask,
			LinkName: model.LinkName,
			Network:  model.Network,
			Mac:      model.Mac,
		})
		if err != nil {
			logrus.Errorf("初始化ip错误 %s", err)
			continue // 初始化失败，跳过该配置，继续处理下一条
		}
	}
}
