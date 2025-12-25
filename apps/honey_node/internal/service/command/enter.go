package command

// File: honey_node/service/command/enter.go
// Description: 节点客户端核心实现，管理与服务端的gRPC连接、命令收发、重连机制及命令分发处理

import (
	"context"
	"honey_node/internal/config"
	"honey_node/internal/rpc/node_rpc"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// NodeClient 节点客户端结构体，负责与服务端的连接管理、命令交互及协程同步
type NodeClient struct {
	client          node_rpc.NodeServiceClient         // gRPC服务客户端实例
	config          *config.Config                     // 节点配置实例
	cmdResponseChan chan *node_rpc.CmdResponse         // 命令响应发送通道
	stream          node_rpc.NodeService_CommandClient // 命令双向流客户端实例
	ctx             context.Context                    // 上下文，用于控制连接生命周期
	cancel          context.CancelFunc                 // 上下文取消函数
	wg              sync.WaitGroup                     // 协程等待组，确保资源优雅释放
	reconnectTimer  *time.Timer                        // 重连计时器
	mu              sync.Mutex                         // 状态保护锁
	isConnected     bool                               // 是否已建立连接的状态标记
	retryDelay      time.Duration                      // 当前重试延迟时间（指数退避）
}

// NewNodeClient 创建节点客户端实例
func NewNodeClient(grpcClient node_rpc.NodeServiceClient,
	config *config.Config) *NodeClient {
	return &NodeClient{
		client:          grpcClient,
		config:          config,
		cmdResponseChan: make(chan *node_rpc.CmdResponse, 10), // 带缓冲的响应通道，容量10
		reconnectTimer:  time.NewTimer(0),                     // 初始化重连定时器（初始状态停止）
		retryDelay:      1 * time.Second,                      // 初始重试延迟1秒
	}
}

// StartCommandHandling 启动命令处理主流程，初始化上下文并建立初始连接
func (nc *NodeClient) StartCommandHandling() {
	// 创建可取消的上下文，用于控制整个命令处理生命周期
	nc.ctx, nc.cancel = context.WithCancel(context.Background())
	nc.wg.Add(1)

	go func() {
		defer nc.wg.Done()

		// 执行初始连接逻辑
		nc.connect()

		// 阻塞等待上下文取消信号（服务停止）
		<-nc.ctx.Done()
		// 清理连接资源
		nc.disconnect()
		logrus.Info("命令处理已停止")
	}()
}

// connect 建立与服务端的命令流连接，包含并发保护
func (nc *NodeClient) connect() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	// 已处于连接状态则直接返回，避免重复连接
	if nc.isConnected {
		return
	}

	// 构造包含节点ID的元数据上下文，用于服务端身份验证
	ctx := metadata.NewOutgoingContext(nc.ctx, metadata.Pairs("nodeID", nc.config.System.Uid))

	// 创建双向命令流
	stream, err := nc.client.Command(ctx)
	if err != nil {
		logrus.Errorf("创建命令流失败: %v，将在%v后重试", err, nc.retryDelay)
		nc.scheduleReconnect(nc.retryDelay)
		nc.increaseRetryDelay()
		return
	}

	// 更新连接状态及流实例
	nc.stream = stream
	nc.isConnected = true
	nc.retryDelay = 1 * time.Second // 连接成功，重置重试延迟
	logrus.Info("节点命令流连接成功")

	// 启动响应发送和请求接收协程
	nc.wg.Add(2)
	go nc.sendResponses()
	go nc.receiveRequests()
}

// disconnect 断开与服务端的连接，清理相关资源
func (nc *NodeClient) disconnect() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	// 未连接状态无需处理
	if !nc.isConnected {
		return
	}

	// 停止重连定时器
	nc.reconnectTimer.Stop()

	// 关闭命令流
	if nc.stream != nil {
		nc.stream.CloseSend()
		nc.stream = nil
	}

	// 关闭并重建响应通道，避免数据残留
	close(nc.cmdResponseChan)
	nc.cmdResponseChan = make(chan *node_rpc.CmdResponse, 10)

	// 更新连接状态标识
	nc.isConnected = false
	logrus.Info("节点命令流已断开")
}

// scheduleReconnect 安排延迟重连任务
// 参数：delay - 重连延迟时间
func (nc *NodeClient) scheduleReconnect(delay time.Duration) {
	// 重置定时器触发时间
	nc.reconnectTimer.Reset(delay)

	go func() {
		// 等待定时器触发
		<-nc.reconnectTimer.C
		// 上下文已取消则不再重连
		if nc.ctx.Err() != nil {
			return
		}
		// 执行重连逻辑
		nc.connect()
	}()
}

// increaseRetryDelay 增加重试延迟时间（指数退避算法）
func (nc *NodeClient) increaseRetryDelay() {
	nc.retryDelay *= 2
	if nc.retryDelay > 60*time.Second {
		nc.retryDelay = 60 * time.Second
	}
}

// sendResponses 循环发送命令响应到服务端
func (nc *NodeClient) sendResponses() {
	defer nc.wg.Done()

	for {
		select {
		// 上下文取消则退出协程
		case <-nc.ctx.Done():
			return

		// 从响应通道读取待发送数据
		case response, ok := <-nc.cmdResponseChan:
			if !ok {
				return
			}

			// 发送响应到服务端
			if err := nc.stream.Send(response); err != nil {
				logrus.Errorf("发送响应失败: %v", err)
				// 发送失败则断开连接并安排重连
				nc.disconnect()
				nc.scheduleReconnect(nc.retryDelay)
				nc.increaseRetryDelay()
				return
			}

			logrus.Debugf("已发送响应: %+v", response)
		}
	}
}

// receiveRequests 循环接收服务端下发的命令请求
func (nc *NodeClient) receiveRequests() {
	defer nc.wg.Done()

	for {
		// 从命令流接收请求
		request, err := nc.stream.Recv()
		if err != nil {
			// 分类处理不同错误类型
			if status.Code(err) == 0 { // io.EOF，服务端主动关闭连接
				logrus.Info("服务器关闭了连接")
			} else if err == context.Canceled { // 上下文取消
				logrus.Info("上下文已取消")
			} else if netErr, ok := err.(net.Error); ok && netErr.Temporary() { // 临时网络错误
				logrus.Warnf("临时网络错误: %v", err)
			} else { // 其他错误
				logrus.Errorf("接收请求失败: %v", err)
			}

			// 断开连接并触发重连
			nc.disconnect()
			nc.scheduleReconnect(nc.retryDelay)
			nc.increaseRetryDelay()
			return
		}

		logrus.Infof("收到命令: %+v", request)
		// 分发处理具体命令
		nc.handleCommand(request)
	}
}

// handleCommand 根据命令类型分发到对应处理方法
// 参数：request - 服务端下发的命令请求
func (nc *NodeClient) handleCommand(request *node_rpc.CmdRequest) {
	switch request.CmdType {
	case node_rpc.CmdType_cmdNetworkFlushType: // 网卡刷新命令
		nc.CmdNetworkFlush(request)
	case node_rpc.CmdType_cmdNetScanType: // 网络扫描命令
		nc.CmdNetScan(request)
	case node_rpc.CmdType_cmdNodeRemoveType: // 节点移除命令
		nc.CmdNodeRemove(request)
	default: // 未知命令类型
		logrus.Warnf("未知命令类型: %v", request.CmdType)
	}
}
