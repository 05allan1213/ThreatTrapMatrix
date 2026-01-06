package port_service

// File: honey_node/service/port_service/tunnel.go
// Description: 端口服务模块，负责本地端口监听管理、TCP隧道创建及与RPC服务的双向数据转发

import (
	"context"
	"errors"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/rpc/node_rpc"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// tunnelEntry 封装单个隧道的 listener 和 cancel 函数，用于代际区分
type tunnelEntry struct {
	listener net.Listener
	cancel   context.CancelFunc
}

// tunnelStore 存储端口监听实例的并发安全映射，key为本地监听地址，value为*tunnelEntry
var tunnelStore = sync.Map{}

// Tunnel 创建本地TCP监听并建立到目标地址的隧道
func Tunnel(localAddr, targetAddr string) (err error) {
	// 检查是否已存在同地址的 listener，如果存在则先关闭旧的（避免竞态）
	if oldValue, exists := tunnelStore.Load(localAddr); exists {
		logrus.Warnf("检测到重复启动 %s，先关闭旧 listener", localAddr)
		oldEntry := oldValue.(*tunnelEntry)
		_ = oldEntry.listener.Close()
		oldEntry.cancel() // 取消旧的所有连接
		// 旧 goroutine 的 defer 会比对指针，不会误删新一代
	}

	// 创建本地TCP监听
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		logrus.Errorf("创建本地监听失败: %v", err)
		// 删除数据库中对应的记录
		global.DB.Where("local_addr = ?", localAddr).Delete(&models.PortModel{})
		return err
	}
	logrus.Infof("本地监听启动，地址: %s", localAddr)
	logrus.Infof("目标地址: %s", targetAddr)

	// 创建 per-listener 的 context，用于统一中止该地址上的所有连接
	listenerCtx, listenerCancel := context.WithCancel(context.Background())

	// 封装为 entry 并存储（用于代际区分）
	entry := &tunnelEntry{
		listener: listener,
		cancel:   listenerCancel,
	}
	tunnelStore.Store(localAddr, entry)

	// 持续接受客户端连接
	go func() {
		defer func() {
			// 只有当前 entry 还在 store 中时才删除（避免误删新一代）
			if current, ok := tunnelStore.Load(localAddr); ok && current.(*tunnelEntry) == entry {
				tunnelStore.Delete(localAddr)
			}
		}()
		defer listenerCancel() // 取消所有子连接

		var acceptBackoff time.Duration // Accept 错误退避时间
		const maxBackoff = time.Second  // 最大退避时间

		for {
			clientConn, err := listener.Accept()
			if err != nil {
				// 使用 errors.Is 判断是否为 listener 关闭错误
				if errors.Is(err, net.ErrClosed) {
					break
				}
				// 其他错误（如 EMFILE 等临时错误）使用指数退避，避免 CPU/日志自旋
				if acceptBackoff == 0 {
					acceptBackoff = 5 * time.Millisecond
				} else {
					acceptBackoff *= 2
					if acceptBackoff > maxBackoff {
						acceptBackoff = maxBackoff
					}
				}
				logrus.Warnf("接受客户端连接失败，%v 后重试: %v", acceptBackoff, err)
				time.Sleep(acceptBackoff)
				continue
			}

			// 成功 Accept，重置退避时间
			acceptBackoff = 0

			// 为每个连接创建一个goroutine处理
			go handleConnection(listenerCtx, global.GrpcClient, clientConn, targetAddr)
		}
	}()
	return nil
}

// handleConnection 处理单个客户端连接的双向数据转发
func handleConnection(parentCtx context.Context, client node_rpc.NodeServiceClient, localConn net.Conn, targetAddr string) {
	ctx, cancel := context.WithCancel(parentCtx)

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
		// 解析 host:port，精确匹配 IP
		host, _, err := net.SplitHostPort(localAddr)
		if err != nil || host != ip {
			return true // 跳过不匹配的地址
		}

		logrus.Infof("清除%s上的全部服务: %s", ip, localAddr)

		// 1. 获取 entry 并关闭监听实例，阻断新连接
		entry := value.(*tunnelEntry)
		_ = entry.listener.Close()

		// 2. 取消该地址上的所有存量连接
		entry.cancel()

		// 3. 清理 tunnelStore 和 DB（只有还是同一代才删除，避免误删并发启动的新代）
		if current, ok := tunnelStore.Load(localAddr); ok && current.(*tunnelEntry) == entry {
			tunnelStore.Delete(localAddr)
			global.DB.Where("local_addr = ?", localAddr).Delete(&models.PortModel{})
		}

		return true
	})
}
