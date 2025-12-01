package mq_service

// File: honey_node/service/mq_service/create_ip_exchange.go
// Description: 创建诱捕IP的MQ消息消费处理逻辑，通过执行系统命令创建macvlan虚拟网络接口并配置IP

import (
	"encoding/json"
	"fmt"
	"honey_node/internal/utils/cmd"
	"strings"

	"github.com/sirupsen/logrus"
)

// CreateIPRequest 创建诱捕IP的消息结构体
type CreateIPRequest struct {
	HoneyIPID uint   `json:"honeyIpID"` // 诱捕ipID
	IP        string `json:"ip"`        // 要配置的诱捕IP地址
	Mask      int8   `json:"mask"`      // 子网掩码位数
	Network   string `json:"network"`   // 绑定的物理网卡名称
	LogID     string `json:"logID"`     // 操作日志ID（用于追踪操作链路）
}

// CreateIpExChange 处理创建诱捕IP的MQ消息，执行系统命令配置macvlan虚拟接口
func CreateIpExChange(msg string) error {
	// 解析MQ消息内容为CreateIPRequest结构体
	var req CreateIPRequest
	err := json.Unmarshal([]byte(msg), &req)
	if err != nil {
		logrus.Errorf("json解析失败 %s %s", err, msg)
		return nil // 返回nil表示消息已消费（即使解析失败，避免重复投递）
	}

	// 构造虚拟网络接口名称（格式：hy_+诱捕IPID，确保唯一性）
	linkName := fmt.Sprintf("hy_%d", req.HoneyIPID)

	// 1. 创建macvlan虚拟网络接口：基于物理接口创建桥接模式的macvlan链路
	cmd.Cmd(fmt.Sprintf("ip link add %s link %s type macvlan mode bridge", linkName, req.Network))
	// 2. 启用虚拟网络接口
	cmd.Cmd(fmt.Sprintf("ip link set %s up", linkName))
	// 3. 为虚拟接口配置IP地址和子网掩码
	cmd.Cmd(fmt.Sprintf("ip addr add %s/%d dev %s", req.IP, req.Mask, linkName))

	// 获取虚拟接口的MAC地址（用于后续上报或存储）
	mac, err := cmd.Command(fmt.Sprintf("ip link show %s | awk '/link\\/ether/ {print $2}'", linkName))
	fmt.Println("mac: ", strings.TrimSpace(mac)) // 打印MAC地址（日志/调试用）

	/* 命令示例参考：
	ip link add mc_12 link ens33 type macvlan mode bridge  # 创建macvlan接口
	ip link set mc_12 up                                   # 启用接口
	ip addr add 192.168.80.166/24 dev mc_12               # 配置IP地址
	*/

	// 调用gRPC方法上报IP创建状态（如MAC地址、接口状态等）

	return nil // 返回nil表示消息处理成功
}
