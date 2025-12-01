package mq_service

// File: honey_node/service/mq_service/bind_port_exchange.go
// Description: 消息队列服务处理模块，负责解析端口绑定相关消息并执行端口隧道管理操作

import (
	"encoding/json"
	"fmt"
	"honey_node/internal/service/port_service"

	"github.com/sirupsen/logrus"
)

// BindPortRequest 端口绑定请求结构体，接收MQ传递的端口绑定参数
type BindPortRequest struct {
	IP       string     `json:"ip"`       // 目标IP地址
	PortList []PortInfo `json:"portList"` // 端口映射列表
	LogID    string     `json:"logID"`    // 日志id，用于链路追踪
}

// PortInfo 端口映射信息结构体，包含本地监听和目标地址信息
type PortInfo struct {
	IP       string `json:"ip"`       // 本地监听IP
	Port     int    `json:"port"`     // 本地监听端口
	DestIP   string `json:"destIP"`   // 目标服务IP
	DestPort int    `json:"destPort"` // 目标服务端口
}

// LocalAddr 拼接本地监听地址（IP:Port）
func (p PortInfo) LocalAddr() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

// TargetAddr 拼接目标服务地址（DestIP:DestPort）
func (p PortInfo) TargetAddr() string {
	return fmt.Sprintf("%s:%d", p.DestIP, p.DestPort)
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
		// 起端口监听，每个端口映射独立协程处理，避免阻塞
		go func(port PortInfo) {
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
