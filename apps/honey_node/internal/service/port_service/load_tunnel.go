package port_service

// File: honey_node/service/port_service/load_tunnel.go
// Description: 端口服务模块，负责应用启动时从数据库加载历史端口转发记录并初始化隧道

import (
	"honey_node/internal/global"
	"honey_node/internal/models"

	"github.com/sirupsen/logrus"
)

// LoadTunnel 应用启动时加载历史端口转发记录，自动初始化所有隧道连接
func LoadTunnel() {
	var portList []models.PortModel
	// 从数据库查询所有端口转发配置记录
	global.DB.Find(&portList)
	logrus.Infof("加载端口转发记录 %d", len(portList))

	// 遍历所有端口转发记录，异步启动隧道（避免阻塞应用启动流程）
	for _, model := range portList {
		go Tunnel(model.LocalAddr, model.TargetAddr)
	}
}
