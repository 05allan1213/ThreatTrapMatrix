package info

// File: honey_node/utils/info/network.go
// Description: 网卡工具包，提供网卡信息的获取与过滤功能

import (
	"net"
	"strings"
)

// NetworkInfo 网卡结构体，存储单个网卡的关键信息
type NetworkInfo struct {
	Network string // 网卡名称
	Ip      string // IP地址（字符串格式）
	Mask    int    // 子网掩码位数
	Net     string // 网络段地址（CIDR格式）
}

// isFilteredNetwork 判断网卡是否应该被过滤
// 支持三种匹配模式：
// 1. 以'-'结尾：前缀匹配
// 2. 以'-'开头：后缀匹配
// 3. 其他：包含匹配
func isFilteredNetwork(networkName string, filterList []string) bool {
	for _, filter := range filterList {
		// 空过滤条件跳过
		if filter == "" {
			continue
		}

		// 根据过滤规则的格式选择不同的匹配方式
		switch {
		case strings.HasSuffix(filter, "-"):
			// 前缀匹配（例如"br-"匹配"br-eth0"）
			prefix := strings.TrimSuffix(filter, "-")
			if strings.HasPrefix(networkName, prefix) {
				return true
			}
		case strings.HasPrefix(filter, "-"):
			// 后缀匹配（例如"-eth"匹配"br-eth"）
			suffix := strings.TrimPrefix(filter, "-")
			if strings.HasSuffix(networkName, suffix) {
				return true
			}
		default:
			// 包含匹配（例如"docker"匹配"docker0"）
			if strings.Contains(networkName, filter) {
				return true
			}
		}
	}
	return false
}

// GetNetworkList 获取系统有效网卡列表，支持过滤指定前缀的网卡
func GetNetworkList(filterNetworkList []string) (list []NetworkInfo, err error) {
	// 获取系统所有网卡
	faces, err := net.Interfaces()
	if err != nil {
		return
	}

	// 遍历每个网卡进行筛选和信息提取
	for _, face := range faces {
		faceName := face.Name
		// 跳过回环接口（lo）
		if faceName == "lo" {
			continue
		}

		// 检查网卡是否应该被过滤
		if isFilteredNetwork(faceName, filterNetworkList) {
			continue
		}

		// 获取当前接口绑定的所有地址
		addrs, err := face.Addrs()
		if err != nil {
			continue
		}
		// 遍历当前接口的每个地址，提取IPv4信息
		for _, addr := range addrs {
			// 解析CIDR格式地址，分离IP和网络段
			ip, _net, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			// 仅保留IPv4地址（过滤IPv6）
			if ip.To4() == nil {
				continue
			}
			// 获取子网掩码位数
			mask, _ := _net.Mask.Size()
			// 组装网卡并添加到结果列表
			list = append(list, NetworkInfo{
				Network: faceName,
				Ip:      ip.String(),
				Mask:    mask,
				Net:     _net.String(),
			})
		}
	}
	return
}