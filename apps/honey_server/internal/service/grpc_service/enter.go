package grpc_service

// File: honey_server/service/grpc_service/enter.go
// Description: gRPC服务端实现，提供节点注册等相关的gRPC服务接口

import (
	"context"
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/rpc/node_rpc"
	"net"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// NodeService 节点服务gRPC实现结构体
type NodeService struct {
	node_rpc.UnimplementedNodeServiceServer // 嵌入未实现的服务端结构体，兼容接口版本
}

// Register 节点注册接口实现
func (NodeService) Register(ctx context.Context, request *node_rpc.RegisterRequest) (pd *node_rpc.BaseResponse, err error) {
	pd = new(node_rpc.BaseResponse) // 初始化响应结构体
	fmt.Println("节点注册", request)
	return
}

// Run 监听指定端口，创建gRPC服务器，注册服务并开始处理客户端请求
func Run() {
	// 从全局配置获取gRPC服务监听地址
	addr := global.Config.System.GrpcAddr
	// 监听TCP端口
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err) // 监听失败则终止程序
	}

	// 创建gRPC服务器实例
	s := grpc.NewServer()
	server := NodeService{}
	// 将NodeService实例注册到gRPC服务器，使其处理对应服务的请求
	node_rpc.RegisterNodeServiceServer(s, &server)
	logrus.Infof("grpc server running %s", addr) // 记录服务启动日志

	// 启动gRPC服务器，开始监听并处理客户端请求（阻塞运行）
	err = s.Serve(listen)
}
