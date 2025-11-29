package main

import (
	"context"
	"fmt"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/ip"
	"os"

	"github.com/google/uuid"
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

	// 采集指定网卡的IPv4地址和MAC地址（网卡名称从配置读取）
	_ip, mac, err := ip.GetNetworkInfo(global.Config.System.Network)
	if err != nil {
		logrus.Fatalln(err) // 网络信息采集失败则终止程序
	}

	// 若配置中无节点唯一标识，生成UUID并持久化到配置文件
	if global.Config.System.Uid == "" {
		global.Config.System.Uid = uuid.New().String()
		core.SetConfig() // 保存配置到文件
	}

	// 获取节点主机名（用于标识节点）
	hostname, err := os.Hostname()
	if err != nil {
		logrus.Fatalln(err) // 获取主机名失败则终止程序
	}

	// 调用管理端Register接口发送节点注册请求
	result, err := client.Register(context.Background(), &node_rpc.RegisterRequest{
		Ip:      _ip,                      // 节点ip
		Mac:     mac,                      // 节点mac
		NodeUid: global.Config.System.Uid, // 节点唯一标识（UUID）
		Version: global.Version,           // 节点版本
		Commit:  global.Commit,            // 节点commit
		SystemInfo: &node_rpc.SystemInfoMessage{
			HostName: hostname, // 节点主机名
		},
	})

	// 打印注册请求结果
	fmt.Println(result, err)
}
