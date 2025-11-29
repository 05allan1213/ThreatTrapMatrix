package node_network_api

// File: honey_server/api/node_network_api/enter.go
// Description: 节点网卡API接口

import "sync"

// NodeNetworkApi 节点网卡API接口结构体
type NodeNetworkApi struct {
	mutex sync.Mutex
}
