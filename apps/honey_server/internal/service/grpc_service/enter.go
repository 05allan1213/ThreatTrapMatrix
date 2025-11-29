package grpc_service

// File: honey_server/service/grpc_service/enter.go
// Description: gRPC服务端实现，提供节点相关的gRPC服务接口

import (
	"crypto/tls"
	"crypto/x509"
	"honey_server/internal/global"
	"honey_server/internal/rpc/node_rpc"
	"io/ioutil"
	"net"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NodeService 节点服务gRPC实现结构体
type NodeService struct {
	node_rpc.UnimplementedNodeServiceServer // 嵌入未实现的服务端结构体，兼容接口版本
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

	// 加载服务端证书和私钥
	cert, err := tls.LoadX509KeyPair("cert/server.crt", "cert/server.key")
	if err != nil {
		logrus.Fatalf("failed to load key pair: %v", err)
	}

	// 加载 CA 证书
	caCert, err := ioutil.ReadFile("cert/ca.crt")
	if err != nil {
		logrus.Fatalf("failed to read CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// 创建 TLS 配置
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert, // 双向认证
		ClientCAs:    caCertPool,
	}

	// 创建 credentials
	creds := credentials.NewTLS(config)

	// 创建 gRPC 服务器，使用 TLS credentials
	s := grpc.NewServer(grpc.Creds(creds))

	server := NodeService{}
	// 将NodeService实例注册到gRPC服务器，使其处理对应服务的请求
	node_rpc.RegisterNodeServiceServer(s, &server)
	logrus.Infof("grpc server running %s", addr) // 记录服务启动日志

	// 启动gRPC服务器，开始监听并处理客户端请求（阻塞运行）
	err = s.Serve(listen)
}
