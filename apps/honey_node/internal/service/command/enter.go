package command

// File: honey_node/service/command/enter.go
// Description: 节点命令客户端模块，管理与服务端的注册、命令交互及断线重连机制

import (
	"context"
	"fmt"
	"honey_node/internal/config"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/info"
	"honey_node/internal/utils/ip"
	"net"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// NodeClient 节点命令客户端，负责与服务端的连接管理、注册及命令交互
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
}

// NewNodeClient 创建节点命令客户端实例
func NewNodeClient(grpcClient node_rpc.NodeServiceClient,
	config *config.Config) *NodeClient {
	return &NodeClient{
		client:          grpcClient,
		config:          config,
		cmdResponseChan: make(chan *node_rpc.CmdResponse, 10), // 带缓冲通道避免发送阻塞
		reconnectTimer:  time.NewTimer(0),
	}
}

// Register 向服务端执行节点注册，上报节点基础信息
func (nc *NodeClient) Register() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取指定网卡的IP和MAC地址
	_ip, mac, err := ip.GetNetworkInfo(nc.config.System.Network)
	if err != nil {
		return fmt.Errorf("获取网络信息失败: %v", err)
	}

	// 获取主机名
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("获取主机名失败: %v", err)
	}

	// 获取系统基础信息（OS版本、内核等）
	systemInfo, err := info.GetSystemInfo()
	if err != nil {
		return fmt.Errorf("获取系统信息失败: %v", err)
	}

	// 获取过滤后的网络接口列表
	networkList, err := nc.getNetworkList("hy-")
	if err != nil {
		return fmt.Errorf("获取网络列表失败: %v", err)
	}

	// 构建注册请求参数
	req := &node_rpc.RegisterRequest{
		Ip:      _ip,
		Mac:     mac,
		NodeUid: nc.config.System.Uid,
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
	_, err = nc.client.Register(ctx, req)
	if err != nil {
		return fmt.Errorf("注册请求失败: %v", err)
	}

	logrus.Infof("节点注册成功，上报信息: %+v", req)
	return nil
}

// StartCommandHandling 启动命令交互主循环，包含断线重连机制
func (nc *NodeClient) StartCommandHandling() {
	nc.ctx, nc.cancel = context.WithCancel(context.Background())
	nc.wg.Add(1)

	go func() {
		defer nc.wg.Done()

		// 初始化建立连接
		nc.connect()

		// 监听上下文取消信号，退出时清理连接
		<-nc.ctx.Done()
		nc.disconnect()
		logrus.Info("命令处理已停止")
	}()
}

// connect 建立与服务端的命令双向流连接
func (nc *NodeClient) connect() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	// 已连接则直接返回
	if nc.isConnected {
		return
	}

	// 创建携带节点ID的元数据上下文
	ctx := metadata.NewOutgoingContext(nc.ctx, metadata.Pairs("nodeID", nc.config.System.Uid))

	// 建立命令双向流连接
	stream, err := nc.client.Command(ctx)
	if err != nil {
		logrus.Errorf("创建命令流失败: %v，将在2秒后重试", err)
		nc.scheduleReconnect(2 * time.Second)
		return
	}

	nc.stream = stream
	nc.isConnected = true
	logrus.Info("节点命令流连接成功")

	// 启动响应发送和请求接收协程
	nc.wg.Add(2)
	go nc.sendResponses()
	go nc.receiveRequests()
}

// disconnect 断开与服务端的命令流连接，清理相关资源
func (nc *NodeClient) disconnect() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if !nc.isConnected {
		return
	}

	// 停止重连计时器
	nc.reconnectTimer.Stop()

	// 关闭双向流
	if nc.stream != nil {
		nc.stream.CloseSend()
		nc.stream = nil
	}

	// 清空并重建响应通道（避免通道关闭后无法复用）
	close(nc.cmdResponseChan)
	nc.cmdResponseChan = make(chan *node_rpc.CmdResponse, 10)

	nc.isConnected = false
	logrus.Info("节点命令流已断开")
}

// scheduleReconnect 安排延迟重连任务
// delay: 重连延迟时间
func (nc *NodeClient) scheduleReconnect(delay time.Duration) {
	nc.reconnectTimer.Reset(delay)

	go func() {
		<-nc.reconnectTimer.C
		// 上下文已取消则不再重连
		if nc.ctx.Err() != nil {
			return
		}
		nc.connect()
	}()
}

// sendResponses 循环读取响应通道并发送至服务端
func (nc *NodeClient) sendResponses() {
	defer nc.wg.Done()

	for {
		select {
		case <-nc.ctx.Done(): // 上下文取消则退出
			return

		case response, ok := <-nc.cmdResponseChan: // 读取待发送的响应
			if !ok { // 通道关闭则退出
				return
			}

			// 发送响应至服务端
			if err := nc.stream.Send(response); err != nil {
				logrus.Errorf("发送响应失败: %v", err)
				nc.disconnect()
				nc.scheduleReconnect(2 * time.Second)
				return
			}

			logrus.Debugf("已发送响应: %+v", response)
		}
	}
}

// receiveRequests 循环接收服务端命令并分发处理
func (nc *NodeClient) receiveRequests() {
	defer nc.wg.Done()

	for {
		// 接收服务端命令请求
		request, err := nc.stream.Recv()
		if err != nil {
			// 分类处理不同错误类型
			if status.Code(err) == 0 { // io.EOF，服务端正常关闭
				logrus.Info("服务器关闭了连接")
			} else if err == context.Canceled { // 上下文取消
				logrus.Info("上下文已取消")
			} else if netErr, ok := err.(net.Error); ok && netErr.Temporary() { // 临时网络错误
				logrus.Warnf("临时网络错误: %v", err)
			} else { // 其他错误
				logrus.Errorf("接收请求失败: %v", err)
			}

			nc.disconnect()
			nc.scheduleReconnect(2 * time.Second)
			return
		}

		logrus.Infof("收到命令: %+v", request)
		// 分发处理具体命令
		nc.handleCommand(request)
	}
}

// handleCommand 根据命令类型分发处理逻辑
// request: 服务端发送的命令请求
func (nc *NodeClient) handleCommand(request *node_rpc.CmdRequest) {
	switch request.CmdType {
	case node_rpc.CmdType_cmdNetworkFlushType: // 网卡刷新命令
		logrus.Info("处理网卡刷新命令")

		// 提取过滤条件（默认空）
		filter := ""
		if request.NetworkFlushInMessage != nil && len(request.NetworkFlushInMessage.FilterNetworkName) > 0 {
			filter = request.NetworkFlushInMessage.FilterNetworkName[0]
		}

		// 获取过滤后的网络列表
		networkList, err := nc.getNetworkList(filter)
		if err != nil {
			logrus.Errorf("获取网络列表失败: %v", err)
			return
		}

		// 构建网卡刷新响应
		response := &node_rpc.CmdResponse{
			CmdType: node_rpc.CmdType_cmdNetworkFlushType,
			TaskID:  request.TaskID,
			NodeID:  nc.config.System.Uid,
			NetworkFlushOutMessage: &node_rpc.NetworkFlushOutMessage{
				NetworkList: networkList,
			},
		}

		// 将响应加入发送队列
		select {
		case nc.cmdResponseChan <- response:
			logrus.Debugf("已将响应加入发送队列: %+v", response)
		case <-nc.ctx.Done(): // 上下文取消则丢弃响应
			logrus.Warn("上下文已取消，丢弃响应")
		}

	default: // 未知命令类型
		logrus.Warnf("未知命令类型: %v", request.CmdType)
	}
}

// getNetworkList 获取系统网络接口列表并转换为RPC消息结构
func (nc *NodeClient) getNetworkList(filter string) ([]*node_rpc.NetworkInfoMessage, error) {
	_networkList, err := info.GetNetworkList(filter)
	if err != nil {
		return nil, err
	}

	var networkList []*node_rpc.NetworkInfoMessage
	// 转换本地网络信息为RPC协议结构
	for _, networkInfo := range _networkList {
		networkList = append(networkList, &node_rpc.NetworkInfoMessage{
			Network: networkInfo.Network,
			Ip:      networkInfo.Ip,
			Net:     networkInfo.Net,
			Mask:    int32(networkInfo.Mask),
		})
	}

	return networkList, nil
}
