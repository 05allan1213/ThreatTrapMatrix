package info

// File: honey_node/utils/info/local_ip.go
// Description: 提供本地IP地址相关工具函数，获取非虚拟接口的IPv4地址映射表，检测指定IP是否为本地网卡绑定的IP（排除hy_前缀的诱捕虚拟接口）

import (
	"net"
	"strings"
	"sync"
)

// LocalIpMap 获取本地非虚拟接口的IPv4地址映射表
func LocalIpMap() map[string]bool {
	var localIpMap = map[string]bool{}
	var mutex sync.Mutex // 互斥锁，保证并发场景下映射表写入的线程安全

	// 获取所有网络接口
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		// 跳过hy_前缀的诱捕虚拟接口（避免检测到部署的诱捕IP）
		if strings.HasPrefix(iface.Name, "hy_") {
			continue
		}

		// 获取当前接口的所有地址
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			// 解析CIDR格式地址，提取纯IP部分
			_ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue // 解析失败则跳过当前地址
			}

			// 仅保留IPv4地址（过滤IPv6）
			if _ip.To4() == nil {
				continue
			}

			// 加锁写入映射表，避免并发写入冲突
			mutex.Lock()
			localIpMap[_ip.String()] = true
			mutex.Unlock()
		}
	}
	return localIpMap
}

// FindLocalIp 检测指定IP是否为本地非虚拟接口绑定的IPv4地址
func FindLocalIp(ip string) bool {
	// 调用LocalIpMap获取本地IP映射表，直接查询指定IP是否存在
	return LocalIpMap()[ip]
}
