package command

// File: honey_node/service/command/register.go
// Description: 节点注册功能实现，完成节点向服务端的身份注册及基础信息上报

import (
	"context"
	"fmt"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/info"
	"honey_node/internal/utils/ip"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Register 向服务端完成节点注册流程，上报节点基础信息、网络信息及系统信息
func (nc *NodeClient) Register() error {
	// 创建带10秒超时的上下文，控制注册请求生命周期
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取指定网卡的IP地址和MAC地址
	_ip, mac, err := ip.GetNetworkInfo(nc.config.System.Network)
	if err != nil {
		return fmt.Errorf("获取网络信息失败: %v", err)
	}

	// 获取节点主机名称
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("获取主机名失败: %v", err)
	}

	// 获取节点系统详细信息（OS版本、内核、架构等）
	systemInfo, err := info.GetSystemInfo()
	if err != nil {
		return fmt.Errorf("获取系统信息失败: %v", err)
	}

	// 获取过滤后的节点网络接口列表
	networkList, err := nc.getNetworkList(nc.config.FilterNetworkList)
	if err != nil {
		return fmt.Errorf("获取网络列表失败: %v", err)
	}

	// 构建注册请求结构体，包含节点身份标识、系统信息及网络信息
	req := &node_rpc.RegisterRequest{
		Ip:      _ip,                  // 节点主IP地址
		Mac:     mac,                  // 节点主网卡MAC地址
		NodeUid: nc.config.System.Uid, // 节点唯一标识
		Version: global.Version,       // 节点程序版本号
		Commit:  global.Commit,        // 节点代码提交哈希
		SystemInfo: &node_rpc.SystemInfoMessage{
			HostName:            hostname,                // 主机名称
			DistributionVersion: systemInfo.OSVersion,    // 操作系统发行版本
			CoreVersion:         systemInfo.Kernel,       // 内核版本
			SystemType:          systemInfo.Architecture, // 系统架构
			StartTime:           systemInfo.BootTime,     // 系统启动时间
		},
		NetworkList: networkList, // 节点网络接口列表
	}

	// 调用服务端注册接口完成注册
	_, err = nc.client.Register(ctx, req)
	if err != nil {
		return fmt.Errorf("注册请求失败: %v", err)
	}

	logrus.Infof("节点注册成功")
	return nil
}
