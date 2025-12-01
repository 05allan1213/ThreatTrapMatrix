package port_service

// File: honey_node/service/port_service/tunnel.go
// Description: 端口服务模块，负责本地端口监听管理、TCP隧道创建及与RPC服务的双向数据转发

import (
	"context"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/rpc/node_rpc"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// tunnelStore 存储端口监听实例的并发安全映射，key为本地监听地址，value为对应的Listener
var tunnelStore = sync.Map{}

// Tunnel 创建本地TCP监听并建立到目标地址的隧道
func Tunnel(localAddr, targetAddr string) (err error) {
	// 创建本地TCP监听
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		logrus.Errorf("创建本地监听失败: %v", err)
		return
	}
	logrus.Infof("本地监听启动，地址: %s", localAddr)
	logrus.Infof("目标地址: %s", targetAddr)
	tunnelStore.Store(localAddr, listener) // 将监听实例存入全局存储

	// 持续接受客户端连接
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "closed") { // 监听被主动关闭时退出循环
				break
			}
			logrus.Errorf("接受客户端连接失败: %v", err)
			break
		}

		// 为每个新连接启动独立协程处理，避免阻塞主监听循环
		go handleConnection(global.GrpcClient, clientConn, targetAddr)
	}
	return nil
}

// handleConnection 处理单个客户端连接的双向数据转发
func handleConnection(client node_rpc.NodeServiceClient, localConn net.Conn, targetAddr string) {
	defer localConn.Close() // 函数退出时关闭本地连接

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保上下文被取消，释放资源

	// 通过RPC创建双向流隧道
	stream, err := client.Tunnel(ctx)
	if err != nil {
		logrus.Infof("创建隧道失败: %v", err)
		return
	}

	// 发送初始隧道配置消息，携带目标地址信息
	if err := stream.Send(&node_rpc.TunnelData{
		Chunk:   []byte{},
		Address: targetAddr,
	}); err != nil {
		logrus.Errorf("发送初始请求失败: %v", err)
		return
	}

	// 启动协程处理RPC服务端到本地连接的数据转发
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF { // 流关闭时退出
				break
			}
			if err != nil {
				logrus.Errorf("接收gRPC服务器数据失败: %v", err)
				break
			}

			// 将RPC接收的数据写入本地客户端连接
			_, err = localConn.Write(resp.Chunk)
			if err != nil {
				logrus.Errorf("写入本地连接失败: %v", err)
				break
			}
		}
		cancel() // 数据转发异常时取消上下文
	}()

	// 处理本地连接到RPC服务端的数据转发
	buffer := make([]byte, 4096) // 4KB缓冲区用于数据读取
	for {
		n, err := localConn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				logrus.Infof("本地连接已关闭")
			} else {
				logrus.Errorf("从本地连接读取失败: %v", err)
			}
			break
		}

		// 将本地读取的数据通过RPC发送到目标服务
		err = stream.Send(&node_rpc.TunnelData{
			Chunk:   buffer[:n],
			Address: targetAddr,
		})
		if err != nil {
			logrus.Errorf("发送数据到gRPC服务器失败: %v", err)
			break
		}
	}

	// 主动关闭RPC发送流
	stream.CloseSend()
}

// CloseIpTunnel 关闭指定IP上的所有端口监听及隧道
func CloseIpTunnel(ip string) {
	// 遍历所有TunnelStore中的隧道
	tunnelStore.Range(func(key, value any) bool {
		localAddr := key.(string)
		// 判断当前隧道的localAddr是否以指定IP开头
		if strings.HasPrefix(localAddr, ip) {
			var model models.PortModel
			global.DB.Find(&model, "local_addr = ?", localAddr)
			if model.ID != 0 {
				global.DB.Delete(&model)
			}
			logrus.Infof("清除%s上的全部服务", ip)
			listener := value.(net.Listener)
			listener.Close() // 关闭监听实例，终止端口服务
		}
		return true
	})
}
