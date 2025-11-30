package ip

// File: honey_node/utils/ip/enter.go
// Description: 网卡工具包，提供获取指定网卡的IPv4地址和MAC地址的功能

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// GetNetworkInfo 获取指定网卡的IPv4地址和MAC地址
func GetNetworkInfo(i string) (ip string, mac string, err error) {
	// 根据网卡名称获取网卡接口信息
	iface, err := net.InterfaceByName(i)
	if err != nil {
		err = fmt.Errorf("无法获取网卡 %s: %v", i, err)
		return
	}

	// 获取该网卡绑定的所有地址（包含IP、子网掩码等信息）
	addrs, err := iface.Addrs()
	if err != nil {
		err = fmt.Errorf("无法获取网卡 %s 的地址: %s", iface.Name, err)
		return
	}

	// 获取网卡的MAC地址并转换为字符串
	mac = iface.HardwareAddr.String()

	// 遍历网卡地址列表，筛选出IPv4地址
	for _, addr := range addrs {
		var _ip net.IP
		// 类型断言区分IPNet（带子网掩码的IP）和IPAddr（纯IP）
		switch v := addr.(type) {
		case *net.IPNet:
			_ip = v.IP
		case *net.IPAddr:
			_ip = v.IP
		}
		// 检查是否为IPv4地址（To4()返回nil表示IPv6）
		if _ip.To4() != nil {
			ip = _ip.String()
		}
	}

	// 校验是否获取到有效IPv4地址
	if ip == "" {
		err = fmt.Errorf("%s 此接口无ip的地址", iface.Name)
		return
	}
	return
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
