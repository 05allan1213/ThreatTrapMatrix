package ip

// File: alert_server/utils/ip/enter.go
// Description: IP地址处理工具类，提供本地IP判断、IP范围解析等网络地址相关操作功能

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// HasLocalIPAddr 判断给定IP地址是否为本地IP（私有地址或回环地址）
func HasLocalIPAddr(_ip string) bool {
	ip := net.ParseIP(_ip)
	// 判断是否为私有地址（RFC1918定义的内网地址）
	if ip.IsPrivate() {
		return true
	}
	// 判断是否为回环地址（127.0.0.0/8或::1/128）
	if ip.IsLoopback() {
		return true
	}
	return false
}

// ParseIPRange 解析IP范围字符串，支持单个IP、IP段（如192.168.1.1-100或192.168.1.1-192.168.1.100）格式
func ParseIPRange(ipRange string) ([]string, error) {
	var result []string
	// 按逗号分割多个IP范围段
	segments := strings.Split(ipRange, ",")

	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}

		// 处理IP段格式（包含连字符）
		if strings.Contains(segment, "-") {
			parts := strings.SplitN(segment, "-", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("无效的IP段格式: %s", segment)
			}

			startIPStr := strings.TrimSpace(parts[0])
			endPart := strings.TrimSpace(parts[1])

			startIP := net.ParseIP(startIPStr)
			if startIP == nil {
				return nil, fmt.Errorf("无效的起始IP: %s", startIPStr)
			}

			// 仅支持IPv4地址段解析
			if ipv4 := startIP.To4(); ipv4 != nil {
				startIP = ipv4
				var endIP net.IP

				// 尝试解析结束部分为完整IP地址
				if endIP = net.ParseIP(endPart); endIP != nil {
					endIP = endIP.To4()
					if endIP == nil {
						return nil, fmt.Errorf("无效的结束IP: %s", endPart)
					}
				} else {
					// 处理简写格式（如192.168.1.1-100），解析最后一个八位组
					endNum, err := strconv.Atoi(endPart)
					if err != nil || endNum < 0 || endNum > 255 {
						return nil, fmt.Errorf("无效的结束部分: %s", endPart)
					}
					// 复制起始IP作为结束IP基础，修改最后一个字节
					endIP = make(net.IP, len(startIP))
					copy(endIP, startIP)
					endIP[len(endIP)-1] = byte(endNum)
				}

				// 遍历生成IP范围内的所有地址
				for cmp := bytes.Compare(startIP, endIP); cmp <= 0; cmp = bytes.Compare(startIP, endIP) {
					result = append(result, startIP.String())
					// 递增IP地址（处理进位）
					for i := len(startIP) - 1; i >= 0; i-- {
						startIP[i]++
						if startIP[i] > 0 {
							break
						}
					}
				}
			} else {
				return nil, fmt.Errorf("IPv6范围解析暂不支持")
			}
		} else {
			// 处理单个IP地址
			ip := net.ParseIP(segment)
			if ip == nil {
				return nil, fmt.Errorf("无效的IP地址: %s", segment)
			}
			result = append(result, ip.String())
		}
	}

	return result, nil
}

// IncrementIP 将IP地址加1，支持IPv4和IPv6地址
func IncrementIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}

	// 复制IP地址，避免修改原IP对象
	newIP := make(net.IP, len(ip))
	copy(newIP, ip)

	// 处理IPv4地址（32位）
	if ip4 := newIP.To4(); ip4 != nil {
		// 从最后一个字节开始逐位递增，遇到非0xFF则停止进位
		for i := 3; i >= 0; i-- {
			newIP[i]++
			if newIP[i] > 0 {
				break
			}
		}
		return newIP
	}

	// 处理IPv6地址（128位）
	// 从最后一个字节开始逐位递增，遇到非0xFF则停止进位
	for i := len(newIP) - 1; i >= 0; i-- {
		newIP[i]++
		if newIP[i] > 0 {
			break
		}
	}
	return newIP
}

// DecrementIP 将IP地址减1，支持IPv4和IPv6地址
func DecrementIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}

	// 复制IP地址，避免修改原IP对象
	newIP := make(net.IP, len(ip))
	copy(newIP, ip)

	// 处理IPv4地址（32位）
	if ip4 := newIP.To4(); ip4 != nil {
		// 从最后一个字节开始逐位递减，遇到非0x00则停止借位
		for i := 3; i >= 0; i-- {
			newIP[i]--
			if newIP[i] < 255 {
				break
			}
		}
		return newIP
	}

	// 处理IPv6地址（128位）
	// 从最后一个字节开始逐位递减，遇到非0x00则停止借位
	for i := len(newIP) - 1; i >= 0; i-- {
		newIP[i]--
		if newIP[i] < 255 {
			break
		}
	}
	return newIP
}

// BroadcastIP 计算IPv4 CIDR网段的广播地址
func BroadcastIP(network *net.IPNet) net.IP {
	ip := network.IP.To4()
	if ip == nil {
		// IPv6无广播地址，返回nil
		return nil
	}

	mask := network.Mask
	result := make(net.IP, len(ip))

	// 按位计算：IP地址 | 子网掩码反码 = 广播地址
	for i := 0; i < len(ip); i++ {
		result[i] = ip[i] | ^mask[i]
	}

	return result
}

// FormatIPRange 将起始和结束IP地址格式化为"起始IP-结束IP"的字符串
func FormatIPRange(start, end net.IP) string {
	return fmt.Sprintf("%s-%s", start, end)
}

// ipToInt 将IPv4地址转换为32位无符号整数
func ipToInt(ip net.IP) uint32 {
	ip4 := ip.To4()
	return (uint32(ip4[0]) << 24) | (uint32(ip4[1]) << 16) | (uint32(ip4[2]) << 8) | uint32(ip4[3])
}

// intToIP 将32位无符号整数转换为IPv4地址字符串
func intToIP(ipInt uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		(ipInt>>24)&0xff, // 取最高8位（第一段）
		(ipInt>>16)&0xff, // 取次高8位（第二段）
		(ipInt>>8)&0xff,  // 取第三段8位
		ipInt&0xff) // 取最低8位（第四段）
}

// ParseCIDRGetUseIPRange 解析CIDR网段，计算可用IP范围（排除网络地址和广播地址）
func ParseCIDRGetUseIPRange(cidr string) (r string, err error) {
	// 解析CIDR字符串为IP对象和网段对象
	ipObj, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		err = errors.New("无效的网段")
		return
	}

	// 获取子网掩码位数（如/24对应24）
	mask, _ := ipNet.Mask.Size()

	// 转换为IPv4地址，确保输入为IPv4网段
	ip4 := ipObj.To4()
	if ip4 == nil {
		err = errors.New("不是有效的IPv4地址")
		return
	}

	// 处理掩码小于24的情况（如/16、/8），自动截取第一个C段
	if mask < 24 {
		// 拆分IP地址为四段字符串
		ipParts := strings.Split(ip4.String(), ".")
		if len(ipParts) != 4 {
			err = errors.New("无效的IPv4地址格式")
			return
		}
		// 构建第一个C段的网络地址（如192.168.0.0/24）
		firstC := fmt.Sprintf("%s.%s.%s.0", ipParts[0], ipParts[1], ipParts[2])
		ip4 = net.ParseIP(firstC).To4()
		mask = 24 // 强制使用24位掩码（C段）
	}

	// 将IPv4地址转换为整数进行计算
	ipInt := ipToInt(ip4)
	maskBits := uint(32 - mask) // 掩码剩余位数（主机位）

	// 计算网络地址整数（主机位清零）
	networkInt := ipInt & (^uint32(0) << maskBits)
	// 计算广播地址整数（主机位全1）
	broadcastInt := networkInt | (^uint32(0) >> (32 - maskBits))

	// 计算可用IP范围：网络地址+1 到 广播地址-1
	firstUsable := networkInt + 1
	lastUsable := broadcastInt - 1

	// 格式化为"起始IP-结束IP"字符串
	r = fmt.Sprintf("%s-%s", intToIP(firstUsable), intToIP(lastUsable))
	return
}
