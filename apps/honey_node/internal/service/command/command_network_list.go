package command

// File: honey_node/service/command/command_network_list.go
// Description: 节点客户端网络信息处理工具方法，实现系统网络列表获取及RPC消息格式转换

import (
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/info"
)

// getNetworkList 获取系统网络接口列表并转换为RPC消息格式
func (nc *NodeClient) getNetworkList(filters []string) ([]*node_rpc.NetworkInfoMessage, error) {
	// 调用系统工具获取过滤后的网络接口信息
	_networkList, err := info.GetNetworkList(filters)
	if err != nil {
		return nil, err
	}

	// 转换系统网络信息为RPC通信所需的消息结构
	var networkList []*node_rpc.NetworkInfoMessage
	for _, networkInfo := range _networkList {
		networkList = append(networkList, &node_rpc.NetworkInfoMessage{
			Network: networkInfo.Network,     // 网卡名称
			Ip:      networkInfo.Ip,          // IP地址
			Net:     networkInfo.Net,         // 网络段
			Mask:    int32(networkInfo.Mask), // 子网掩码位数
		})
	}

	return networkList, nil
}
