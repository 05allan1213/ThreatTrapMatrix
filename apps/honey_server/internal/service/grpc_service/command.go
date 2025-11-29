package grpc_service

// File: honey_server/service/grpc_service/command.go
// Description: 处理节点命令交互与双向流通信，管理命令请求/响应通道

import (
	"fmt"
	"honey_server/internal/rpc/node_rpc"
	"io"

	"github.com/sirupsen/logrus"
)

// CmdRequestChan 命令请求通道，用于传递发给节点的命令请求
var CmdRequestChan = make(chan *node_rpc.CmdRequest, 0)

// CmdResponseChan 命令响应通道，用于接收节点返回的命令执行结果
var CmdResponseChan = make(chan *node_rpc.CmdResponse, 0)

// Command 实现NodeService的双向流Command接口，处理节点与服务端的命令交互
// stream: gRPC双向流通信的服务端流对象
func (NodeService) Command(stream node_rpc.NodeService_CommandServer) error {
	// 启动goroutine持续接收节点推送的响应数据
	go func() {
		for {
			response, err := stream.Recv()
			if err == io.EOF {
				logrus.Infof("节点断开")
				return
			}
			if err != nil {
				logrus.Infof("节点出错 %s", err)
				return
			}
			// 接收节点返回的命令执行结果并转发至响应通道
			fmt.Println("命令结果", response)
			CmdResponseChan <- response
		}
	}()

	// 监听请求通道，将待发送的命令请求推送至节点
	for request := range CmdRequestChan {
		err := stream.Send(request)
		if err != nil {
			logrus.Infof("数据发送失败 %s", err)
			continue
		}
	}
	return nil
}
