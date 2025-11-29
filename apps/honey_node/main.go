package main

import (
	"context"
	"fmt"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/service/cron_service"
	"honey_node/internal/utils/info"
	"honey_node/internal/utils/ip"
	"io"
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
	// 初始化gRPC客户端连接
	global.GrpcClient = core.GetGrpcClient()

	// 执行节点注册流程
	err := register()
	if err != nil {
		logrus.Errorf("节点注册失败 %s", err)
		return
	}
	logrus.Infof("节点注册成功")

	// 启动命令交互协程
	go command()
	// 启动定时任务服务
	cron_service.Run()
	// 通过空select阻塞主线程，防止程序退出（定时任务为后台goroutine，需主线程存活）
	select {}
}

// CmdResponseChan 命令响应通道，用于缓存待发送给服务端的命令执行结果
var CmdResponseChan = make(chan *node_rpc.CmdResponse, 0)

// command 处理节点与服务端的命令交互，实现双向流通信逻辑
func command() {
	// 建立Command双向流连接
	stream, err := global.GrpcClient.Command(context.Background())
	if err != nil {
		if err != nil { // 原逻辑保留：重复错误判断
			logrus.Errorf("节点Command失败 %s", err)
			return
		}
	}

	// 启动协程持续接收服务端命令请求
	go func() {
		for {
			request, err := stream.Recv()
			if err == io.EOF {
				logrus.Infof("节点断开")
				return
			}
			if err != nil {
				logrus.Infof("节点出错 %s", err)
				return
			}
			fmt.Println("接收管理的数据", request)

			// 根据命令类型分发处理逻辑
			switch request.CmdType {
			case node_rpc.CmdType_cmdNetworkFlushType:
				fmt.Println("网卡刷新")
				// 获取过滤后的网络接口列表
				_networkList, _ := info.GetNetworkList(request.NetworkFlushInMessage.FilterNetworkName[0])
				var networkList []*node_rpc.NetworkInfoMessage
				// 转换为RPC协议定义的消息结构
				for _, networkInfo := range _networkList {
					networkList = append(networkList, &node_rpc.NetworkInfoMessage{
						Network: networkInfo.Network,
						Ip:      networkInfo.Ip,
						Net:     networkInfo.Net,
						Mask:    int32(networkInfo.Mask),
					})
				}
				// 组装网卡刷新响应并发送至通道
				CmdResponseChan <- &node_rpc.CmdResponse{
					CmdType: node_rpc.CmdType_cmdNetworkFlushType,
					TaskID:  "xx",
					NodeID:  global.Config.System.Uid,
					NetworkFlushOutMessage: &node_rpc.NetworkFlushOutMessage{
						NetworkList: networkList,
					},
				}
			}
		}
	}()

	// 监听响应通道，将结果发送至服务端
	for response := range CmdResponseChan {
		err := stream.Send(response)
		if err != nil {
			logrus.Infof("数据发送失败 %s", err)
			continue
		}
	}
}

// register 完成节点向服务端的注册流程，上报节点基础信息
func register() (err error) {
	// 获取指定网卡的IP和MAC地址
	_ip, mac, err := ip.GetNetworkInfo(global.Config.System.Network)
	if err != nil {
		return
	}

	// 获取主机名
	hostname, err := os.Hostname()
	if err != nil {
		return
	}
	// 获取系统信息（OS版本、内核等）
	systemInfo, err := info.GetSystemInfo()
	if err != nil {
		return
	}

	// 获取过滤后的网络接口列表（排除hy-后缀网卡）
	var networkList []*node_rpc.NetworkInfoMessage
	_networkList, err := info.GetNetworkList("hy-")
	if err != nil {
		return
	}
	// 转换网络信息为RPC消息结构
	for _, networkInfo := range _networkList {
		networkList = append(networkList, &node_rpc.NetworkInfoMessage{
			Network: networkInfo.Network,
			Ip:      networkInfo.Ip,
			Net:     networkInfo.Net,
			Mask:    int32(networkInfo.Mask),
		})
	}

	// 组装注册请求参数
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
		NetworkList: networkList,
	}

	// 发送注册请求至服务端
	_, err = global.GrpcClient.Register(context.Background(), &req)
	if err != nil {
		logrus.Fatalf("节点注册失败 %s", err)
		return
	}
	logrus.Infof("节点注册 上报信息 %v", req)
	return nil
}
