package main

import (
	"context"
	"fmt"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 读取系统配置文件
	global.Config = core.ReadConfig()
	// 设置日志默认配置
	core.SetLogDefault()
	// 初始化全局日志实例
	global.Log = core.GetLogger()

	// 从配置获取gRPC管理服务地址
	addr := global.Config.System.GrpcManageAddr

	// 创建gRPC客户端连接
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		// 连接失败时记录致命日志并终止程序
		logrus.Fatalf(fmt.Sprintf("grpc connect addr [%s] 连接失败 %s", addr, err))
	}
	defer conn.Close() // 延迟关闭连接，确保程序退出时释放资源

	// 初始化节点服务gRPC客户端实例
	client := node_rpc.NewNodeServiceClient(conn)
	// 调用管理端Register接口发送节点注册请求
	result, err := client.Register(context.Background(), &node_rpc.RegisterRequest{
		Ip:      "",    // 节点IP
		Mac:     "xx",  // 节点MAC地址
		NodeUid: "xxx", // 节点唯一标识
		Version: "",    // 节点程序版本
		Commit:  "",    // 节点Commit
	})

	// 打印注册请求结果
	fmt.Println(result, err)
}
