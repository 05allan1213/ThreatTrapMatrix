package core

// File: honey_node/core/grpc_client.go
// Description: gRPC客户端管理核心组件，封装gRPC连接与客户端实例的创建逻辑，提供统一的客户端获取入口

import (
	"honey_node/internal/global"
	"honey_node/internal/rpc"
	"honey_node/internal/rpc/node_rpc"
)

// GetGrpcClient 获取节点服务的gRPC客户端实例
func GetGrpcClient() node_rpc.NodeServiceClient {
	// 从全局配置中读取管理端gRPC服务地址
	addr := global.Config.System.GrpcManageAddr

	// 获取gRPC连接
	conn := rpc.GetConn(addr)

	// 基于gRPC连接初始化节点服务客户端实例
	client := node_rpc.NewNodeServiceClient(conn)

	return client
}
