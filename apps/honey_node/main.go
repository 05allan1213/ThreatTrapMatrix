package main

import (
	"context"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/service/command"
	"honey_node/internal/service/cron_service"
	"honey_node/internal/service/mq_service"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

// nodeClient 全局节点客户端实例，管理与服务端的交互
var nodeClient *command.NodeClient

func main() {
	// 加载系统配置文件
	global.Config = core.ReadConfig()
	// 设置日志默认配置
	core.SetLogDefault()
	// 初始化全局日志实例
	global.Log = core.GetLogger()

	// 创建gRPC客户端连接
	global.GrpcClient = core.GetGrpcClient()

	// 初始化节点命令客户端实例
	nodeClient = command.NewNodeClient(global.GrpcClient, global.Config)

	// 执行节点注册流程
	if err := nodeClient.Register(); err != nil {
		logrus.Fatalf("节点注册失败: %v", err)
		return
	}

	// 初始化rabbitMQ连接
	global.Queue = core.InitMQ()

	// 启动命令处理服务（接收并处理服务端下发的命令）
	nodeClient.StartCommandHandling()

	// 启动定时任务
	cron_service.Run()
	// 启动rabbitMQ消费者
	mq_service.Run()

	// 启动TCP监听服务（用于端口转发隧道）
	go tcpListen()

	// 阻塞主协程，保持程序运行
	select {}
}

// 隧道配置：本地监听地址与目标转发地址
var localAddr = "192.168.5.130:8081" // 节点本地监听地址（接收外部请求）
var targetAddr = "127.0.0.1:8080"    // 隧道目标地址（服务端转发的目标）

// tcpListen 启动TCP本地监听服务
// 接收外部TCP连接，并通过gRPC隧道转发到服务端指定目标
func tcpListen() {
	// 创建TCP监听（绑定本地地址）
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("创建本地监听失败: %v", err)
	}
	defer listener.Close() // 程序退出时关闭监听

	log.Printf("本地监听启动，地址: %s", localAddr)
	log.Printf("目标地址: %s", targetAddr)

	// 信号处理：捕获中断/终止信号，实现优雅关闭
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint
		log.Println("接收到终止信号，优雅关闭...")
		os.Exit(0)
	}()

	// 循环接收客户端连接（阻塞等待）
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("接受客户端连接失败: %v", err)
			continue
		}

		// 为每个连接启动独立协程处理（避免阻塞监听）
		go handleConnection(global.GrpcClient, clientConn, targetAddr)
	}
}

// handleConnection 处理单个TCP连接的隧道转发逻辑
func handleConnection(client node_rpc.NodeServiceClient, localConn net.Conn, targetAddr string) {
	defer localConn.Close() // 函数退出时关闭本地连接

	// 创建带取消功能的上下文（用于控制gRPC流生命周期）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 函数退出时取消上下文

	// 创建gRPC双向流（Tunnel接口）：用于节点与服务端的双向数据传输
	stream, err := client.Tunnel(ctx)
	if err != nil {
		log.Printf("创建隧道失败: %v", err)
		return
	}

	// 发送隧道初始化消息：告知服务端目标转发地址
	if err := stream.Send(&node_rpc.TunnelData{
		Chunk:   []byte{},   // 初始空数据
		Address: targetAddr, // 目标转发地址
	}); err != nil {
		log.Printf("发送初始请求失败: %v", err)
		return
	}

	// 协程1：gRPC 流 -> 本地 TCP (下行流量)
	// 读取gRPC服务端数据并写入本地连接
	go func() {
		for {
			// 从gRPC流接收服务端转发的数据
			resp, err := stream.Recv()
			if err == io.EOF {
				break // 流关闭，退出循环
			}
			if err != nil {
				log.Printf("接收gRPC服务器数据失败: %v", err)
				break
			}

			// 将服务端数据写入本地TCP连接（转发给外部客户端）
			_, err = localConn.Write(resp.Chunk)
			if err != nil {
				log.Printf("写入本地连接失败: %v", err)
				break
			}
		}
		cancel() // 数据传输异常，取消上下文
	}()

	// 协程2：本地 TCP -> gRPC 流 (上行流量)
	// 读取本地连接数据并发送到gRPC服务端
	buffer := make([]byte, 4096) // 数据缓冲区（4KB）
	for {
		// 从本地TCP连接读取外部客户端的数据
		n, err := localConn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Println("本地连接已关闭")
			} else {
				log.Printf("从本地连接读取失败: %v", err)
			}
			break
		}

		// 将本地数据发送到gRPC服务端（转发到目标地址）
		err = stream.Send(&node_rpc.TunnelData{
			Chunk:   buffer[:n], // 实际读取的数据
			Address: targetAddr, // 目标转发地址
		})
		if err != nil {
			log.Printf("发送数据到gRPC服务器失败: %v", err)
			break
		}
	}

	// 关闭gRPC流的发送端（告知服务端数据传输完成）
	stream.CloseSend()
}
