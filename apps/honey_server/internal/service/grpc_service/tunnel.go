package grpc_service

// File: honey_server/service/grpc_service/tunnel.go
// Description: 节点与服务端之间的TCP隧道实现，使用gRPC进行数据转发

import (
	"fmt"
	"honey_server/internal/rpc/node_rpc"
	"io"
	"log"
	"net"
)

// Tunnel 实现node_rpc.NodeServiceServer接口的双向流Tunnel方法
func (s *NodeService) Tunnel(stream node_rpc.NodeService_TunnelServer) error {
	// 接收客户端的第一个消息（初始化消息）：获取隧道目标地址
	req, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("接收初始请求失败: %v", err)
	}

	// 建立与目标地址的TCP连接（使用流上下文控制超时/取消）
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(stream.Context(), "tcp", req.Address)
	if err != nil {
		return fmt.Errorf("连接目标地址失败: %v", err)
	}
	defer conn.Close() // 函数退出时关闭目标连接

	// 协程1：处理 gRPC 流 -> TCP 连接 (上行流量)
	// 读取gRPC客户端（节点）数据并转发到目标TCP连接
	go func() {
		for {
			// 从gRPC流接收节点发送的数据
			req, err := stream.Recv()
			if err == io.EOF {
				return // 客户端流关闭，退出协程
			}
			if err != nil {
				log.Printf("接收客户端数据失败: %v", err)
				return
			}

			// 将节点数据写入目标TCP连接（转发到目标服务）
			_, err = conn.Write(req.Chunk)
			if err != nil {
				log.Printf("写入目标连接失败: %v", err)
				return
			}
		}
	}()

	// 协程2（主逻辑）：处理 TCP 连接 -> gRPC 流 (下行流量)
	// 读取目标TCP连接数据并转发到gRPC客户端（节点）
	buffer := make([]byte, 4096) // 4KB数据缓冲区，平衡IO效率与内存占用
	for {
		// 从目标TCP连接读取数据
		n, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Println("目标连接已关闭")
			} else {
				log.Printf("从目标连接读取失败: %v", err)
			}
			return nil // 目标连接关闭/异常，结束隧道
		}

		// 将目标数据通过gRPC流发送给节点客户端
		err = stream.Send(&node_rpc.TunnelData{
			Chunk:   buffer[:n],  // 实际读取的有效数据
			Address: req.Address, // 目标地址（保持上下文）
		})
		if err != nil {
			log.Printf("发送数据到客户端失败: %v", err)
			return err // 发送失败，返回错误终止隧道
		}
	}
}
