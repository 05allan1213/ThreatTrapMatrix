package main

import (
	"honey_node/internal/core"
	"honey_node/internal/flags"
	"honey_node/internal/global"
	"honey_node/internal/service/command"
	"honey_node/internal/service/cron_service"
	"honey_node/internal/service/mq_service"
	"honey_node/internal/service/port_service"

	"github.com/sirupsen/logrus"
)

// 全局节点客户端实例
var nodeClient *command.NodeClient

func main() {
	// 加载系统配置文件
	global.Config = core.ReadConfig()
	// 设置日志默认配置
	core.SetLogDefault()
	// 初始化全局日志实例
	global.Log = core.GetLogger()
	// 初始化数据库连接
	global.DB = core.GetDB()

	// 创建gRPC客户端连接
	global.GrpcClient = core.GetGrpcClient()

	// 初始化节点客户端
	nodeClient = command.NewNodeClient(global.GrpcClient, global.Config)

	// 执行节点注册流程
	if err := nodeClient.Register(); err != nil {
		logrus.Fatalf("节点注册失败: %v", err)
		return
	}

	// 运行命令行参数处理
	flags.Run()

	// 初始化rabbitMQ连接
	global.Queue = core.InitMQ()

	// 启动命令处理服务（接收并处理服务端下发的命令）
	nodeClient.StartCommandHandling()

	// 启动定时任务
	cron_service.Run()

	// 启动rabbitMQ消费者
	mq_service.Run()

	// 加载tunnel
	port_service.LoadTunnel()

	// 阻塞主协程，保持程序运行
	select {}
}
