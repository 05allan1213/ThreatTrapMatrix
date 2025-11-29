package global

// File: honey_node/global/enter.go
// Description: 全局变量模块，定义应用程序级别的全局共享变量

import (
	"honey_node/internal/config"
	"honey_node/internal/rpc/node_rpc"

	"github.com/sirupsen/logrus"
)

// 全局变量声明区
var (
	Config     *config.Config             // 全局配置实例
	Log        *logrus.Entry              // 全局日志实例
	GrpcClient node_rpc.NodeServiceClient // 全局gRPC客户端实例
)

var (
	Version   = "v1.0.1"
	Commit    = "a29bb955"
	BuildTime = "2025-11-24 19:45:58"
)
