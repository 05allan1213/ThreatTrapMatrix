package ip_service

// File: honey_node/service/ip_service/set_ip.go
// Description: 网络接口配置模块，负责MACVLAN接口创建、IP地址配置、MAC地址设置及相关网络操作，支持失败自动资源清理

import (
	"fmt"
	"honey_node/internal/utils/cmd"
	"strings"

	"github.com/sirupsen/logrus"
)

// SetIpRequest 网络接口配置请求结构体，包含创建MACVLAN接口及配置IP所需参数
type SetIpRequest struct {
	Ip       string `json:"ip"`       // 待配置的IP地址
	Mask     int8   `json:"mask"`     // 子网掩码
	LinkName string `json:"linkName"` // 目标网络接口名称（待创建的MACVLAN接口名）
	Network  string `json:"network"`  // 基础网卡名称（基于该网卡创建MACVLAN接口）
	Mac      string `json:"mac"`      // MAC地址
}

// SetIp 执行网络接口配置流程：创建MACVLAN接口→（可选）设置MAC地址→启用接口→添加IP地址
func SetIp(req SetIpRequest) (mac string, err error) {
	linkName := req.LinkName
	// 资源清理函数：当配置过程中出现错误时，删除已创建的网络接口，避免残留垃圾资源
	cleanup := func() {
		if err := cmd.Cmd(fmt.Sprintf("ip link delete %s", linkName)); err != nil {
			logrus.Errorf("清理失败，删除网络接口 %s 时出错: %v", linkName, err)
		}
	}

	// 1. 基于基础网卡创建MACVLAN接口（桥接模式）
	if err = createMacVlanInterface(linkName, req.Network); err != nil {
		logrus.Errorf("创建macvlan接口失败: %v", err)
		cleanup()
		return
	}

	// 2. 若指定了MAC地址，为接口设置自定义MAC
	if req.Mac != "" {
		err = setInterfaceMac(linkName, req.Mac)
		if err != nil {
			logrus.Errorf("设置mac失败: %v", err)
			cleanup()
			return
		}
	}

	// 3. 启用网络接口（接口默认处于禁用状态，需手动启用）
	if err = setInterfaceUp(linkName); err != nil {
		logrus.Errorf("启用网络接口失败: %v", err)
		cleanup()
		return
	}

	// 4. 为接口添加指定IP地址及子网掩码
	if err = addIPAddress(linkName, req.Ip, req.Mask); err != nil {
		logrus.Errorf("添加IP地址失败: %v", err)
		cleanup()
		return
	}

	// 5. 若未指定MAC地址，获取系统自动分配的MAC地址
	if req.Mac == "" {
		req.Mac, err = GetMACAddress(linkName)
		if err != nil {
			return
		}
	}

	return req.Mac, nil
}

// createMacVlanInterface 基于指定基础网卡创建MACVLAN接口，采用桥接模式
func createMacVlanInterface(linkName, network string) error {
	cmdStr := fmt.Sprintf("ip link add %s link %s type macvlan mode bridge", linkName, network)
	if err := cmd.Cmd(cmdStr); err != nil {
		return fmt.Errorf("执行命令失败 [%s]: %w", cmdStr, err)
	}
	return nil
}

// setInterfaceUp 启用指定网络接口（将接口状态设置为UP）
func setInterfaceUp(linkName string) error {
	cmdStr := fmt.Sprintf("ip link set %s up", linkName)
	if err := cmd.Cmd(cmdStr); err != nil {
		return fmt.Errorf("执行命令失败 [%s]: %w", cmdStr, err)
	}
	return nil
}

// setInterfaceMac 为指定网络接口设置自定义MAC地址
func setInterfaceMac(linkName string, mac string) error {
	cmdStr := fmt.Sprintf("ip link set %s address %s", linkName, mac)
	if err := cmd.Cmd(cmdStr); err != nil {
		return fmt.Errorf("执行命令失败 [%s]: %w", cmdStr, err)
	}
	return nil
}

// addIPAddress 为指定网络接口添加IP地址及子网掩码
func addIPAddress(linkName, ip string, mask int8) error {
	cmdStr := fmt.Sprintf("ip addr add %s/%d dev %s", ip, mask, linkName)
	if err := cmd.Cmd(cmdStr); err != nil {
		return fmt.Errorf("执行命令失败 [%s]: %w", cmdStr, err)
	}
	return nil
}

// GetMACAddress 获取指定网络接口的MAC地址
func GetMACAddress(linkName string) (string, error) {
	// 通过ip link命令结合awk提取MAC地址（匹配link/ether后的字段）
	cmdStr := fmt.Sprintf("ip link show %s | awk '/link\\/ether/ {print $2}'", linkName)
	mac, err := cmd.Command(cmdStr)
	if err != nil {
		return "", fmt.Errorf("执行命令失败 [%s]: %w", cmdStr, err)
	}
	return strings.TrimSpace(mac), nil
}
