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
		// 删除数据库中对应的记录
		var model models.PortModel
		global.DB.Where("local_addr = ?", localAddr).First(&model)
		if model.ID != 0 {
			global.DB.Delete(&model)
		}
		return err
	}
	logrus.Infof("本地监听启动，地址: %s", localAddr)
	logrus.Infof("目标地址: %s", targetAddr)
	tunnelStore.Store(localAddr, listener) // 将监听实例存入全局存储

	// 持续接受客户端连接
	go func() {
		for {
			clientConn, err := listener.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "closed") {
					break
				}
				logrus.Errorf("接受客户端连接失败: %v", err)
				break
			}

			// 为每个连接创建一个goroutine处理
			go handleConnection(global.GrpcClient, clientConn, targetAddr)
		}
	}()
	return nil
}

// handleConnection 处理单个客户端连接的双向数据转发
func handleConnection(client node_rpc.NodeServiceClient, localConn net.Conn, targetAddr string) {
	ctx, cancel := context.WithCancel(context.Background())

	var stream node_rpc.NodeService_TunnelClient

	// 定义幂等的关闭函数，用于在任意错误或取消时打断阻塞的 localConn.Read()
	var closeOnce sync.Once
	closeAll := func() {
		closeOnce.Do(func() {
			_ = localConn.Close() // 关闭本地连接
			if stream != nil {
				_ = stream.CloseSend() // 关闭双向流
			}
		})
	}
	defer closeAll() // 函数退出时关闭连接
	defer cancel()   // 确保上下文被取消，释放资源

	// 通过RPC创建双向流隧道
	var err error
	stream, err = client.Tunnel(ctx)
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

	// 用于等待两个goroutine完成的WaitGroup
	var wg sync.WaitGroup
	wg.Add(2)

	// 启动 watcher：当上下文取消/超时/对端断流时，主动关闭连接打断阻塞读
	go func() {
		<-ctx.Done()
		closeAll()
	}()

	// 协程1：处理 gRPC 流 -> 本地连接 (下行流量)
	// 从gRPC服务端接收数据并转发到本地客户端连接
	go func() {
		defer wg.Done()
		defer closeAll() // 下行退出时关闭连接，打断上行的 localConn.Read()
		defer cancel()   // 取消上下文，通知其他goroutine退出

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 从gRPC流接收服务端发送的数据
				resp, err := stream.Recv()
				if err != nil {
					// 任意接收错误（含 io.EOF）都退出
					if ctx.Err() == nil {
						logrus.Errorf("接收gRPC服务器数据失败: %v", err)
					}
					return
				}

				// 将接收的数据写入本地客户端连接
				_, err = localConn.Write(resp.Chunk)
				if err != nil {
					if ctx.Err() == nil {
						logrus.Errorf("写入本地连接失败: %v", err)
					}
					return
				}
			}
		}
	}()

	// 协程2：处理 本地连接 -> gRPC 流 (上行流量)
	// 从本地客户端连接读取数据并转发到gRPC服务端
	go func() {
		defer wg.Done()
		defer closeAll() // 上行退出时关闭连接，打断下行的 localConn.Read()
		defer cancel()   // 取消上下文，通知其他goroutine退出

		buffer := make([]byte, 4096) // 4KB缓冲区，平衡IO效率与内存占用
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 从本地客户端连接读取数据
				n, err := localConn.Read(buffer)
				if err != nil {
					// 区分"主动关闭"与"真实IO错误"
					if err != io.EOF && ctx.Err() == nil {
						logrus.Errorf("从本地连接读取失败: %v", err)
					}
					return
				}

				// 将本地读取的数据通过gRPC流发送到服务端
				err = stream.Send(&node_rpc.TunnelData{
					Chunk:   buffer[:n],
					Address: targetAddr,
				})
				if err != nil {
					if ctx.Err() == nil {
						logrus.Errorf("发送数据到gRPC服务器失败: %v", err)
					}
					return
				}
			}
		}
	}()

	// 等待两个goroutine都完成后，再关闭stream
	wg.Wait()
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
