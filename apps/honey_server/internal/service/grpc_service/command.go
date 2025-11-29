package grpc_service

// File: honey_server/service/grpc_service/command.go
// Description: 处理节点命令交互与双向流通信，管理命令请求/响应通道

import (
	"errors"
	"fmt"
	"honey_server/internal/rpc/node_rpc"
	"io"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

type Command struct {
	ReqChan chan *node_rpc.CmdRequest
	ResChan chan *node_rpc.CmdResponse
	Server  node_rpc.NodeService_CommandServer
}

var NodeCommandMap = map[string]*Command{}

// Command 实现NodeService的双向流Command接口，处理节点与服务端的命令交互
// stream: gRPC双向流通信的服务端流对象
func (NodeService) Command(stream node_rpc.NodeService_CommandServer) error {
	ctx := stream.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	nodeIDList := md.Get("nodeID")
	if len(nodeIDList) == 0 {
		return errors.New("请在metadata中传入节点id")
	}
	nodeID := nodeIDList[0]
	NodeCommandMap[nodeID] = &Command{
		ReqChan: make(chan *node_rpc.CmdRequest),
		ResChan: make(chan *node_rpc.CmdResponse),
		Server:  stream,
	}
	// 启动goroutine持续接收节点推送的响应数据
	go func() {
		for request := range NodeCommandMap[nodeID].ReqChan {
			err := NodeCommandMap[nodeID].Server.Send(request)
			if err != nil {
				logrus.Infof("数据发送失败 %s", err)
				continue
			}
		}
	}()
	for {
		response, err := NodeCommandMap[nodeID].Server.Recv()
		if err == io.EOF {
			logrus.Infof("节点断开")
			break
		}
		if err != nil {
			logrus.Infof("节点出错 %s", err)
			break
		}
		// 节点往管理发的，命令的执行结果
		fmt.Println("命令结果", response)
		NodeCommandMap[nodeID].ResChan <- response
	}

	delete(NodeCommandMap, nodeID)
	return nil
}
