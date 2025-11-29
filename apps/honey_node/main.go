package main

import (
	"context"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/service/cron_service"
	"honey_node/internal/utils/info"
	"honey_node/internal/utils/ip"
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	// 读取系统配置文件
	global.Config = core.ReadConfig()
	// 设置日志默认配置
	core.SetLogDefault()
	// 初始化全局日志实例
	global.Log = core.GetLogger()
	// 创建gRPC客户端实例
	global.GrpcClient = core.GetGrpcClient()

	// 节点注册
	err := register()
	if err != nil {
		logrus.Errorf("节点注册失败 %s", err)
		return
	}
	logrus.Infof("节点注册成功")

	// 启动定时任务服务
	cron_service.Run()

	// 通过空select阻塞主线程，防止程序退出（定时任务为后台goroutine，需主线程存活）
	select {}
}

// register 节点注册核心函数
func register() (err error) {
	// 采集指定网卡的IPv4地址和MAC地址（网卡名称从配置读取）
	_ip, mac, err := ip.GetNetworkInfo(global.Config.System.Network)
	if err != nil {
		return
	}

	// 获取节点主机名（用于标识节点身份）
	hostname, err := os.Hostname()
	if err != nil {
		return
	}

	// 采集系统详细信息（发行版本、内核、架构、启动时间）
	systemInfo, err := info.GetSystemInfo()
	if err != nil {
		return
	}

	// 构造节点注册请求（映射采集到的信息到gRPC请求结构体）
	req := node_rpc.RegisterRequest{
		Ip:      _ip,
		Mac:     mac,
		NodeUid: global.Config.System.Uid,
		Version: global.Version,
		Commit:  global.Commit,
		SystemInfo: &node_rpc.SystemInfoMessage{
			HostName:            hostname,
			DistributionVersion: systemInfo.OSVersion,
			CoreVersion:         systemInfo.Kernel,
			SystemType:          systemInfo.Architecture,
			StartTime:           systemInfo.BootTime,
		},
	}

	// 调用gRPC注册接口完成节点注册
	_, err = global.GrpcClient.Register(context.Background(), &req)
	if err != nil {
		logrus.Fatalf("节点注册失败 %s", err)
		return
	}

	// 记录注册成功日志
	logrus.Infof("节点注册 上报信息 %v", req)
	return nil
}
