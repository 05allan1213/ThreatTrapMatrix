package info

// File: honey_node/utils/info/network_map.go
// Description: 系统信息工具模块，提供网络接口相关信息查询能力，核心功能为获取本机所有启用状态的网卡及对应IPv4地址

import (
	"fmt"
	"net"
)

// GetNetworkInterfaces 获取本机所有启用状态（UP）的网络接口及其对应的IPv4地址列表
func GetNetworkInterfaces() (map[string][]string, error) {
	// interfacesMap 存储网卡名称与对应IPv4地址列表的映射关系
	interfacesMap := make(map[string][]string)

	// 获取本机所有网络接口（包括物理网卡、虚拟网卡等）
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("获取网络接口失败: %v", err)
	}

	// 遍历每个网络接口，筛选有效IPv4地址
	for _, iface := range interfaces {
		// 忽略状态未启用（非UP）的接口，仅处理正常工作的网卡
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 获取当前接口绑定的所有网络地址（包含IPv4、IPv6等）
		addresses, err := iface.Addrs()
		if err != nil {
			fmt.Printf("获取接口 %s 的地址失败: %v\n", iface.Name, err)
			continue // 单个接口获取失败不影响整体，继续处理下一个接口
		}

		// 筛选当前接口的IPv4地址，排除IPv6及其他类型地址
		var ipv4Addresses []string
		for _, addr := range addresses {
			var ip net.IP

			// 处理不同类型的网络地址（IPNet包含子网掩码信息，IPAddr仅包含IP）
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// 验证是否为有效IPv4地址（To4()返回nil表示非IPv4）
			if ip != nil && ip.To4() != nil {
				ipv4Addresses = append(ipv4Addresses, ip.String())
			}
		}

		// 仅将包含IPv4地址的接口信息存入结果映射
		if len(ipv4Addresses) > 0 {
			interfacesMap[iface.Name] = ipv4Addresses
		}
	}

	return interfacesMap, nil
}
