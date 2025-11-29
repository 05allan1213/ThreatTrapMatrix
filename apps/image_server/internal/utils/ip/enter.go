package ip

// File: image_server/utils/ip/enter.go
// Description: 提供IP地址相关的工具方法，包含本地IP地址判断功能

import "net"

// HasLocalIPAddr 判断给定的IP地址是否为本地/私有IP地址
func HasLocalIPAddr(_ip string) bool {
	// 解析字符串格式的IP地址为net.IP类型
	ip := net.ParseIP(_ip)
	// 判断是否为私有IP地址
	if ip.IsPrivate() {
		return true
	}
	// 判断是否为回环地址
	if ip.IsLoopback() {
		return true
	}
	// 非本地IP地址返回false
	return false
}
