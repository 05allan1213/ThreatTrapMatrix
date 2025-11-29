package ip

// File: honey_node/utils/ip/enter.go
// Description: 网卡工具包，提供获取指定网卡的IPv4地址和MAC地址的功能

import (
	"fmt"
	"net"
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
