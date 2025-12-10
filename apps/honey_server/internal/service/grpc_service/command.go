package grpc_service

// File: honey_server/service/grpc_service/command.go
// Description: 节点gRPC命令服务管理模块，处理节点双向流通信、命令分发及连接状态管理

import (
	"errors"
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/service/mq_service"
	"io"
	"sync"
	"time"

	"honey_server/internal/rpc/node_rpc"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

// Command 单个节点的命令交互实例，管理节点的请求/响应通道及流连接
type Command struct {
	ReqChan  chan *node_rpc.CmdRequest          // 发送给节点的请求通道
	ResChan  chan *node_rpc.CmdResponse         // 节点返回的响应通道
	Server   node_rpc.NodeService_CommandServer // 节点的gRPC双向流服务端实例
	NodeID   string                             // 节点唯一标识
	stopChan chan struct{}                      // 停止信号通道，用于协程退出控制
	wg       sync.WaitGroup                     // 协程等待组，确保所有协程正常退出
	mu       sync.RWMutex                       // 实例状态保护锁
	closed   bool                               // 实例是否已关闭的状态标记
}

var (
	NodeCommandMap = make(map[string]*Command) // 节点命令实例映射表，nodeID -> Command实例
	mapMutex       sync.RWMutex                // 映射表读写保护锁
)

// Command 实现NodeService的双向流Command接口，处理节点连接与命令交互生命周期
func (s NodeService) Command(stream node_rpc.NodeService_CommandServer) error {
	ctx := stream.Context()
	// 从上下文提取元数据（包含节点标识）
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errors.New("missing metadata")
	}

	// 获取节点ID元数据
	nodeIDList := md.Get("nodeID")
	if len(nodeIDList) == 0 {
		return errors.New("nodeID not found in metadata")
	}
	nodeID := nodeIDList[0]

	// 初始化节点命令交互实例
	cmd := &Command{
		ReqChan:  make(chan *node_rpc.CmdRequest, 10), // 带缓冲通道避免发送阻塞
		ResChan:  make(chan *node_rpc.CmdResponse, 10),
		Server:   stream,
		NodeID:   nodeID,
		stopChan: make(chan struct{}),
	}

	// 安全将实例添加到全局映射表
	mapMutex.Lock()
	NodeCommandMap[nodeID] = cmd
	mapMutex.Unlock()

	logrus.Infof("Node %s connected", nodeID)

	// 修改节点状态
	var model models.NodeModel
	err := global.DB.Take(&model, "uid = ?", nodeID).Error
	if err != nil {
		logrus.Errorf("节点不存在")
		return nil
	}
	if model.Status != 1 {
		global.DB.Model(&model).Update("status", 1)
	}

	mq_service.SendWsMsg(mq_service.WsMsgType{
		Type:   4,
		NodeID: model.ID,
	}) // 节点上线

	// 启动发送和接收协程（需等待两个协程退出）
	cmd.wg.Add(2)
	go cmd.sendLoop()
	go cmd.receiveLoop()

	// 监听上下文取消事件（连接断开时触发）
	go func() {
		<-ctx.Done()
		logrus.Infof("Context cancelled for node %s", nodeID)
		cmd.Close()
	}()

	// 等待所有协程完成工作
	cmd.wg.Wait()

	// 从全局映射表移除节点实例
	mapMutex.Lock()
	delete(NodeCommandMap, nodeID)
	mapMutex.Unlock()

	logrus.Infof("Node %s disconnected", nodeID)
	// 修改节点状态
	global.DB.Model(&model).Update("status", 2)
	mq_service.SendWsMsg(mq_service.WsMsgType{
		Type:   4,
		NodeID: model.ID,
	}) // 节点离线
	return nil
}

// sendLoop 循环读取请求通道并发送至节点的协程方法
func (c *Command) sendLoop() {
	defer c.wg.Done() // 协程结束时通知WaitGroup

	for {
		select {
		case req, ok := <-c.ReqChan:
			if !ok { // 请求通道已关闭
				logrus.Infof("ReqChan closed for node %s", c.NodeID)
				return
			}

			// 发送请求至节点
			err := c.Server.Send(req)
			if err != nil {
				logrus.Errorf("Failed to send to node %s: %v", c.NodeID, err)
				c.Close()
				return
			}

		case <-c.stopChan: // 收到停止信号
			logrus.Infof("Stopping sendLoop for node %s", c.NodeID)
			return
		}
	}
}

// receiveLoop 循环接收节点响应并写入通道的协程方法
func (c *Command) receiveLoop() {
	defer c.wg.Done() // 协程结束时通知WaitGroup

	for {
		// 从节点流接收响应
		res, err := c.Server.Recv()
		if err != nil {
			if err == io.EOF { // 节点正常断开
				logrus.Infof("Node %s disconnected normally", c.NodeID)
			} else { // 接收异常
				logrus.Errorf("Receive error from node %s: %v", c.NodeID, err)
			}
			c.Close()
			return
		}

		// 带超时发送响应至通道，防止阻塞
		select {
		case c.ResChan <- res:
			logrus.Debugf("Sent response from node %s to channel", c.NodeID)
		case <-time.After(5 * time.Second): // 5秒超时丢弃
			logrus.Warnf("Timeout sending response from node %s, dropping", c.NodeID)
		case <-c.stopChan: // 收到停止信号
			logrus.Infof("Stopping receiveLoop for node %s", c.NodeID)
			return
		}
	}
}

// Close 关闭节点命令实例，释放资源并终止协程
func (c *Command) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed { // 防止重复关闭
		return
	}

	logrus.Infof("Closing command channels for node %s", c.NodeID)
	c.closed = true

	// 发送停止信号并关闭请求通道
	close(c.stopChan)
	close(c.ReqChan)

	// 异步清空响应通道（避免阻塞）
	go func() {
		for range c.ResChan {
		}
	}()
}

// GetNodeCommand 根据节点ID获取命令交互实例
func GetNodeCommand(nodeID string) (*Command, bool) {
	mapMutex.RLock()
	defer mapMutex.RUnlock()

	cmd, ok := NodeCommandMap[nodeID]
	return cmd, ok
}
