package grpc_service

// File: honey_server/service/grpc_service/tunnel.go
// Description: 节点与服务端之间的TCP隧道实现，使用gRPC进行数据转发

import (
	"errors"
	"fmt"
	"honey_server/internal/rpc/node_rpc"
	"io"
	"log"
	"net"
	"sync"
)

// Tunnel 实现node_rpc.NodeServiceServer接口的双向流Tunnel方法
func (s *NodeService) Tunnel(stream node_rpc.NodeService_TunnelServer) error {
	// 接收客户端的第一个消息（初始化消息）：获取隧道目标地址
	initReq, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("接收初始请求失败: %v", err)
	}

	// 获取流上下文，用于控制整个隧道生命周期
	ctx := stream.Context()

	// 建立与目标地址的TCP连接（使用流上下文控制超时/取消）
	dialer := &net.Dialer{}
	targetConn, err := dialer.DialContext(ctx, "tcp", initReq.Address)
	if err != nil {
		return fmt.Errorf("连接目标地址失败: %v", err)
	}

	// 定义幂等的连接关闭函数，用于在任意错误或取消时打断阻塞的 targetConn.Read()
	closeOnce := sync.Once{}
	closeConn := func() {
		closeOnce.Do(func() {
			_ = targetConn.Close()
		})
	}
	defer closeConn() // 函数退出时关闭目标连接

	// 启动 watcher：当上下文取消/超时/对端断流时，主动关闭连接打断阻塞读
	go func() {
		<-ctx.Done()
		closeConn()
	}()

	// 协程1：处理 gRPC 流 -> TCP 连接 (上行流量)
	// 读取gRPC客户端（节点）数据并转发到目标TCP连接
	go func() {
		defer closeConn() // 上行退出时关闭连接，打断下行的 targetConn.Read()
		for {
			// 从gRPC流接收节点发送的数据
			req, err := stream.Recv()
			if err != nil {
				// 任意接收错误（含 io.EOF）都关闭连接并退出
				closeConn()
				return
			}

			// 将节点数据写入目标TCP连接（转发到目标服务）
			_, err = targetConn.Write(req.Chunk)
			if err != nil {
				// 写入失败时关闭连接并退出
				closeConn()
				return
			}
		}
	}()

	// 协程2（主逻辑）：处理 TCP 连接 -> gRPC 流 (下行流量)
	// 读取目标TCP连接数据并转发到gRPC客户端（节点）
	buffer := make([]byte, 4096) // 4KB数据缓冲区，平衡IO效率与内存占用
	for {
		// 从目标TCP连接读取数据
		n, err := targetConn.Read(buffer)
		if err != nil {
			// 区分"主动关闭"与"真实IO错误"
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				// 上下文取消或连接被我们主动关闭，正常退出
				return nil
			}
			// 真实IO错误（如目标服务断开），记录并退出
			if err != io.EOF {
				log.Printf("从目标连接读取失败: %v", err)
			}
			return nil
		}

		// 将目标数据通过gRPC流发送给节点客户端
		err = stream.Send(&node_rpc.TunnelData{
			Chunk:   buffer[:n],      // 实际读取的有效数据
			Address: initReq.Address, // 目标地址（保持上下文）
		})
		if err != nil {
			// 区分"上下文取消"与"真实发送错误"
			if ctx.Err() != nil {
				// 上下文已取消，正常退出
				return nil
			}
			// 真实发送错误，关闭连接并返回错误
			log.Printf("发送数据到客户端失败: %v", err)
			closeConn()
			return err
		}
	}
}
