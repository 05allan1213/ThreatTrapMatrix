package mq_service

// File: honey_node/service/mq_service/create_ip_exchange.go
// Description: 创建诱捕IP的MQ消息消费处理逻辑，集成ARP检测IP占用、macvlan虚拟接口配置、资源自动清理及gRPC状态上报功能

import (
	"context"
	"encoding/json"
	"fmt"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/cmd"
	"net"
	"strings"

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

	// 探针ip处理
	if req.IsTan {
		mac, _ := getMACAddress(req.Network)
		return reportStatus(req.HoneyIPID, req.Network, mac, "")
	}

	// 记录处理开始日志（包含全链路追踪字段）
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

	// 资源清理函数：当配置过程中出现错误时，清理已创建的虚拟接口
	cleanup := func() {
		if err := cmd.Cmd(fmt.Sprintf("ip link delete %s", linkName)); err != nil {
			logrus.Errorf("清理失败，删除网络接口 %s 时出错: %v", linkName, err)
		}
	}

	// 分步执行macvlan接口配置，每步失败均触发资源清理并上报状态
	if err := createMacVlanInterface(linkName, req.Network); err != nil {
		logrus.Errorf("创建macvlan接口失败: %v", err)
		cleanup()
		return reportStatus(req.HoneyIPID, linkName, "", err.Error())
	}

	if err := setInterfaceUp(linkName); err != nil {
		logrus.Errorf("启用网络接口失败: %v", err)
		cleanup()
		return reportStatus(req.HoneyIPID, linkName, "", err.Error())
	}

	if err := addIPAddress(linkName, req.IP, req.Mask); err != nil {
		logrus.Errorf("添加IP地址失败: %v", err)
		cleanup()
		return reportStatus(req.HoneyIPID, linkName, "", err.Error())
	}

	// 获取虚拟接口的MAC地址（用于上报及后续管理）
	mac, err := getMACAddress(linkName)
	if err != nil {
		logrus.Errorf("获取MAC地址失败: %v", err)
		cleanup()
		return reportStatus(req.HoneyIPID, linkName, "", err.Error())
	}

	// 所有步骤成功，上报成功状态
	return reportStatus(req.HoneyIPID, linkName, mac, "")
}

// createMacVlanInterface 创建macvlan虚拟网络接口（桥接模式）
func createMacVlanInterface(linkName, network string) error {
	cmdStr := fmt.Sprintf("ip link add %s link %s type macvlan mode bridge", linkName, network)
	if err := cmd.Cmd(cmdStr); err != nil {
		return fmt.Errorf("执行命令失败 [%s]: %w", cmdStr, err)
	}
	return nil
}

// setInterfaceUp 启用指定的网络接口
func setInterfaceUp(linkName string) error {
	cmdStr := fmt.Sprintf("ip link set %s up", linkName)
	if err := cmd.Cmd(cmdStr); err != nil {
		return fmt.Errorf("执行命令失败 [%s]: %w", cmdStr, err)
	}
	return nil
}

// addIPAddress 为网络接口配置IP地址和子网掩码
func addIPAddress(linkName, ip string, mask int8) error {
	cmdStr := fmt.Sprintf("ip addr add %s/%d dev %s", ip, mask, linkName)
	if err := cmd.Cmd(cmdStr); err != nil {
		return fmt.Errorf("执行命令失败 [%s]: %w", cmdStr, err)
	}
	return nil
}

// getMACAddress 获取指定网络接口的MAC地址
func getMACAddress(linkName string) (string, error) {
	cmdStr := fmt.Sprintf("ip link show %s | awk '/link\\/ether/ {print $2}'", linkName)
	mac, err := cmd.Command(cmdStr)
	if err != nil {
		return "", fmt.Errorf("执行命令失败 [%s]: %w", cmdStr, err)
	}
	return strings.TrimSpace(mac), nil // 去除MAC地址前后空白字符
}

// reportStatus 通过gRPC向服务端上报IP创建状态（成功/失败）
func reportStatus(honeyIPID uint, network, mac, errMsg string) error {
	response, err := global.GrpcClient.StatusCreateIP(context.Background(), &node_rpc.StatusCreateIPRequest{
		HoneyIPID: uint32(honeyIPID), // 诱捕ipID
		ErrMsg:    errMsg,            // 错误详情
		Network:   network,           // 创建的虚拟接口名称
		Mac:       mac,               // 接口MAC地址
	})

	if err != nil {
		logrus.Errorf("上报管理状态失败: %v", err)
		return err // 返回错误表示消息处理失败（触发MQ重新投递）
	}

	// 记录上报成功日志（包含关键结果字段）
	logrus.WithFields(logrus.Fields{
		"honeyIPID": honeyIPID,
		"network":   network,
		"mac":       mac,
		"errMsg":    errMsg,
	}).Infof("上报管理状态成功: %v", response)

	return nil
}
