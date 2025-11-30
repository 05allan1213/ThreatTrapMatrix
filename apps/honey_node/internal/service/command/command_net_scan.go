package command

// File: honey_node/service/command/command_net_scan.go
// Description: 节点客户端网络扫描命令实现，基于ARP协议进行IP存活探测，支持并发扫描及进度反馈

import (
	"fmt"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/ip"
	"net"
	"sync"
	"time"

	"github.com/j-keck/arping"
)

// CmdNetScan 执行网络扫描任务，通过ARP协议探测指定IP范围内的存活主机，支持并发控制和实时进度反馈
func (nc *NodeClient) CmdNetScan(request *node_rpc.CmdRequest) {
	// 提取网络扫描请求参数
	req := request.GetNetScanInMessage()
	fmt.Printf("开始执行网络扫描任务: %v\n", req)
	startTime := time.Now()

	// 解析扫描IP范围字符串为具体IP列表
	ipList, err := ip.ParseIPRange(req.IpRange)
	if err != nil {
		fmt.Println("IP范围解析错误:", err)
		// 发送扫描失败响应
		nc.cmdResponseChan <- &node_rpc.CmdResponse{
			CmdType: node_rpc.CmdType_cmdNetScanType,
			TaskID:  request.TaskID,
			NodeID:  nc.config.System.Uid,
			NetScanOutMessage: &node_rpc.NetScanOutMessage{
				End:      true,
				Progress: 0,
				NetID:    req.NetID,
				ErrMsg:   fmt.Sprintf("解析扫描ip列表出错 %s", err),
			},
		}
		return
	}

	// 构建过滤IP集合，用于跳过无需扫描的IP
	filterIPList := map[string]struct{}{}
	for _, s := range req.FilterIPList {
		filterIPList[s] = struct{}{}
	}

	iface := req.Network          // 扫描使用的网络接口名称
	concurrency := 200            // 最大并发扫描数（控制扫描速率）
	totalIPs := len(ipList)       // 待扫描IP总数
	processed := 0                // 已处理IP计数
	var processedMutex sync.Mutex // 保护processed变量的并发安全锁

	// 创建信号量通道，实现并发控制
	semaphore := make(chan struct{}, concurrency)

	fmt.Printf("开始扫描 %d 个IP地址，并发数: %d\n", totalIPs, concurrency)

	var wg sync.WaitGroup // 等待组，用于等待所有扫描协程完成
	for _, ipAddr := range ipList {
		// 跳过过滤列表中的IP
		if _, exists := filterIPList[ipAddr]; exists {
			continue
		}

		wg.Add(1)
		// 获取信号量（达到并发上限时阻塞）
		semaphore <- struct{}{}

		// 启动协程执行ARP扫描
		go func(targetIP string) {
			defer wg.Done() // 协程结束时标记等待组任务完成
			defer func() {
				<-semaphore // 释放信号量，允许新协程执行
			}()

			// 通过ARP协议探测目标IP是否存活，获取MAC地址
			macAddr, _, err := arping.PingOverIfaceByName(net.ParseIP(targetIP), iface)

			// 更新扫描进度（加锁保证并发安全）
			processedMutex.Lock()
			processed++
			progress := float64(processed) / float64(totalIPs) * 100 // 计算进度百分比
			processedMutex.Unlock()

			// 目标IP无响应则直接返回
			if err != nil {
				return
			}

			fmt.Printf("探测到存活主机: %s %s 进度: %.2f%%\n", targetIP, macAddr, progress)
			// 发送存活主机探测结果响应
			nc.cmdResponseChan <- &node_rpc.CmdResponse{
				CmdType: node_rpc.CmdType_cmdNetScanType,
				TaskID:  request.TaskID,
				NodeID:  nc.config.System.Uid,
				NetScanOutMessage: &node_rpc.NetScanOutMessage{
					End:      false,
					Progress: float32(progress),
					NetID:    req.NetID,
					Ip:       targetIP,
					Mac:      macAddr.String(),
				},
			}
		}(ipAddr) // 传入当前IP地址（避免循环变量引用问题）
	}

	// 等待所有扫描协程完成
	wg.Wait()
	close(semaphore) // 关闭信号量通道

	// 发送扫描完成最终响应
	nc.cmdResponseChan <- &node_rpc.CmdResponse{
		CmdType: node_rpc.CmdType_cmdNetScanType,
		TaskID:  request.TaskID,
		NodeID:  nc.config.System.Uid,
		NetScanOutMessage: &node_rpc.NetScanOutMessage{
			End:      true,
			Progress: 100,
			NetID:    req.NetID,
		},
	}

	fmt.Printf("\n扫描完成，总耗时: %v\n", time.Since(startTime))
}
