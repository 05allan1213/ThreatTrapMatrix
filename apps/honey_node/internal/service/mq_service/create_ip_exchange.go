package mq_service

// File: honey_node/service/mq_service/create_ip_exchange.go
// Description: 创建诱捕IP的MQ消息消费处理逻辑，集成ARP检测IP占用、macvlan虚拟接口配置、资源自动清理及gRPC状态上报功能

import (
	"context"
	"encoding/json"
	"fmt"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/service/ip_service"
	"net"

	"github.com/j-keck/arping"
	"github.com/sirupsen/logrus"
)

// CreateIPRequest 创建诱捕IP的消息结构体
type CreateIPRequest struct {
	HoneyIPID uint   `json:"honeyIpID"` // 诱捕ipID（用于命名虚拟网络接口及关联数据库）
	IP        string `json:"ip"`        // 待创建的诱捕IP地址
	Mask      int8   `json:"mask"`      // 子网掩码位数（如24）
	Network   string `json:"network"`   // 绑定的物理网卡名称
	LogID     string `json:"logID"`     // 操作日志ID（用于追踪操作链路）
	IsTan     bool   `json:"isTan"`     // 是否是探针ip
}

// CreateIpExChange 处理创建诱捕IP的MQ消息，包含ARP预检测、macvlan配置、资源清理及状态上报
func CreateIpExChange(msg string) error {
	var req CreateIPRequest
	// 解析MQ消息为结构体
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		logrus.Errorf("JSON解析失败: %v, 消息: %s", err, msg)
		return nil // 解析失败返回nil，避免消息重复投递
	}

	// 探针IP特殊处理：无需创建接口，直接获取基础网卡MAC并上报状态
	if req.IsTan {
		mac, _ := ip_service.GetMACAddress(req.Network)
		return reportStatus(req.HoneyIPID, req.Network, mac, "")
	}

	// 记录IP创建请求处理日志，携带核心业务参数
	global.Log.WithFields(logrus.Fields{
		"honeyIPID": req.HoneyIPID,
		"ip":        req.IP,
		"mask":      req.Mask,
		"network":   req.Network,
		"logID":     req.LogID,
	}).Info("开始处理创建IP请求")

	// ARP预检测：检查目标IP是否已被局域网内其他设备占用
	_mac, _, err := arping.PingOverIfaceByName(net.ParseIP(req.IP), req.Network)
	if err == nil {
		// IP已被占用，直接上报失败状态
		err = fmt.Errorf("创建诱捕ip失败 ip已存在 ip %s mac %s", req.IP, _mac.String())
		logrus.Error(err)
		return reportStatus(req.HoneyIPID, "", _mac.String(), err.Error())
	}

	// 构造虚拟网络接口名称（格式：hy_+诱捕IPID，确保唯一性）
	linkName := fmt.Sprintf("hy_%d", req.HoneyIPID)

	// 调用IP服务创建MACVLAN接口并配置IP
	mac, err := ip_service.SetIp(ip_service.SetIpRequest{
		Ip:       req.IP,
		Mask:     req.Mask,
		LinkName: linkName,
		Network:  req.Network,
	})
	if err != nil {
		// 接口创建失败，上报错误状态
		return reportStatus(req.HoneyIPID, linkName, mac, err.Error())
	}

	// IP配置成功，持久化到数据库（支持应用重启后自动恢复配置）
	global.DB.Create(&models.IpModel{
		Ip:       req.IP,
		Mask:     req.Mask,
		LinkName: linkName,
		Network:  req.Network,
		Mac:      mac,
	})

	// 所有步骤执行成功，上报成功状态
	return reportStatus(req.HoneyIPID, linkName, mac, "")
}

// reportStatus 向管理服务上报IP创建结果状态
func reportStatus(honeyIPID uint, network, mac, errMsg string) error {
	// 调用RPC接口上报状态
	response, err := global.GrpcClient.StatusCreateIP(context.Background(), &node_rpc.StatusCreateIPRequest{
		HoneyIPID: uint32(honeyIPID),
		ErrMsg:    errMsg,
		Network:   network,
		Mac:       mac,
	})

	if err != nil {
		logrus.Errorf("上报管理状态失败: %v", err)
		return err
	}

	// 记录状态上报成功日志，携带关键参数便于排查
	logrus.WithFields(logrus.Fields{
		"honeyIPID": honeyIPID,
		"network":   network,
		"mac":       mac,
		"errMsg":    errMsg,
	}).Infof("上报管理状态成功: %v", response)

	return nil
}
