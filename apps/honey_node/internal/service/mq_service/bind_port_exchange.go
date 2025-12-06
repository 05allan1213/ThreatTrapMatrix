package mq_service

// File: honey_node/service/mq_service/bind_port_exchange.go
// Description: 消息队列服务处理模块，负责解析端口绑定相关消息并执行端口隧道管理操作

import (
	"encoding/json"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/port_service"
	"net"

	"github.com/sirupsen/logrus"
)

// BindPortRequest 端口绑定请求结构体，接收MQ传递的端口绑定参数
type BindPortRequest struct {
	IP       string            `json:"ip"`       // 目标IP地址
	PortList []models.PortInfo `json:"portList"` // 端口映射列表
	LogID    string            `json:"logID"`    // 日志id，用于链路追踪
}

// isLocalAddress 检查指定的IP地址是否在本地系统中存在
func isLocalAddress(ip string) bool {
	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		logrus.Errorf("获取网络接口失败: %v", err)
		return false
	}

	// 遍历所有网络接口
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// 检查每个地址
		for _, addr := range addrs {
			var ipAddr net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ipAddr = v.IP
			case *net.IPAddr:
				ipAddr = v.IP
			}

			// 如果找到匹配的IP地址，返回true
			if ipAddr != nil && ipAddr.String() == ip {
				return true
			}
		}
	}

	return false
}

// BindPortExChange 处理端口绑定消息，解析后执行端口隧道创建逻辑
func BindPortExChange(msg string) error {
	logrus.Infof("端口绑定消息 %#v", msg)
	var req BindPortRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		logrus.Errorf("JSON解析失败: %v, 消息: %s", err, msg)
		return nil // 保持原有逻辑，解析失败返回nil
	}
	// 先把之前这个ip上的服务全部停止，避免端口占用冲突
	port_service.CloseIpTunnel(req.IP)

	for _, port := range req.PortList {
		// 检查本地是否存在该IP地址
		if !isLocalAddress(port.IP) {
			logrus.Warnf("本地不存在IP地址 %s，跳过端口绑定", port.IP)
			continue
		}

		// 起端口监听，每个端口映射独立协程处理，避免阻塞
		global.DB.Create(&models.PortModel{
			TargetAddr: port.TargetAddr(),
			LocalAddr:  port.LocalAddr(),
		})
		go func(port models.PortInfo) {
			err := port_service.Tunnel(port.LocalAddr(), port.TargetAddr())
			if err != nil {
				logrus.Errorf("端口绑定失败 %s", err)
			}
			// 如果报错，大概率是ip没有起来，也可能是端口没有释放掉
			// 需要通知管理，只通知失败的
		}(port)
	}

	return nil
}
