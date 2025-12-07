package mq_service

// File: honey_node/service/mq_service/bind_port_exchange.go
// Description: 节点端口绑定MQ消息处理模块，消费端口绑定指令消息，执行端口隧道创建、端口记录持久化，并将绑定结果上报至管理服务的gRPC接口

import (
	"context"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/service/port_service"

	"github.com/sirupsen/logrus"
)

// BindPortRequest 端口绑定请求结构体，接收MQ传递的端口绑定参数
type BindPortRequest struct {
	IP        string            `json:"ip"`        // 待绑定端口的目标IP地址
	HoneyIpID uint              `json:"honeyIpID"` // 关联的诱捕IP ID，用于上报状态时关联管理服务的诱捕IP记录
	PortList  []models.PortInfo `json:"portList"`  // 端口配置列表，包含待绑定的端口号、本地/目标地址等信息
	LogID     string            `json:"logID"`     // 日志ID，用于关联端口绑定操作的全链路日志
}

// BindPortExChange 消费端口绑定MQ消息的核心处理函数
func BindPortExChange(req BindPortRequest) error {
	log := core.GetLogger().WithField("logID", req.LogID)
	log.WithField("req_data", req).Infof("port binding message") // 端口绑定信息

	// 第一步：停止目标IP上已有的所有端口隧道，避免端口占用冲突
	port_service.CloseIpTunnel(req.IP)

	// 初始化端口绑定状态列表，用于收集绑定结果上报至管理服务
	var portInfoList []*node_rpc.StatusPortInfo

	// 第二步：遍历端口配置列表，逐个执行端口隧道绑定
	for _, port := range req.PortList {
		// 持久化端口配置记录到数据库
		global.DB.Create(&models.PortModel{
			TargetAddr: port.TargetAddr(), // 端口转发的目标地址
			LocalAddr:  port.LocalAddr(),  // 本地监听地址
		})

		// 初始化端口状态信息（默认无错误）
		portInfo := &node_rpc.StatusPortInfo{
			Port: int64(port.Port), // 待绑定的端口号
		}

		// 执行端口隧道绑定（创建端口转发）
		err := port_service.Tunnel(port.LocalAddr(), port.TargetAddr())
		if err != nil {
			// 绑定失败时记录错误日志，并填充端口状态的错误信息
			log.WithField("error", err).Errorf("failed to bind port") // 端口绑定失败
			portInfo.Msg = err.Error()
			// 绑定失败原因：大概率是IP未启用/端口未释放，需仅上报失败状态至管理服务
		}

		// 将端口状态加入上报列表
		portInfoList = append(portInfoList, portInfo)
	}

	// 第三步：将端口绑定结果上报至管理服务的gRPC接口
	reportBindPortStatus(req.HoneyIpID, portInfoList, req.LogID)
	return nil
}

// reportBindPortStatus 上报端口绑定状态至管理服务
func reportBindPortStatus(honeyIPID uint, portInfoList []*node_rpc.StatusPortInfo, logID string) error {
	log := core.GetLogger().WithField("logID", logID)
	data := &node_rpc.StatusBindPortRequest{
		HoneyIPID:    uint32(honeyIPID), // 转换为uint32适配gRPC参数类型
		PortInfoList: portInfoList,
		LogID:        logID,
	}
	_, err := global.GrpcClient.StatusBindPort(context.Background(), data)
	if err != nil {
		// 上报失败记录错误日志
		log.WithField("error", err).Errorf("failed to report the management status") // 上报管理状态失败
		return err
	}

	// 上报成功记录结构化日志
	log.WithFields(logrus.Fields{
		"data": data,
	}).Infof("report the management status successfully") // 上报管理状态成功
	return nil
}
