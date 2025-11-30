package command

// File: honey_node/service/command/command_net_scan.go
// Description: 节点客户端网络扫描命令实现，基于ARP协议进行IP存活探测，支持并发扫描及进度反馈

import (
	"fmt"
	"honey_node/internal/core"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/ip"
	"net"
	"sync"
	"time"

	"github.com/j-keck/arping"
)

// CmdNetScan 处理网络扫描命令请求，执行ARP扫描并返回扫描结果
func (nc *NodeClient) CmdNetScan(request *node_rpc.CmdRequest) {
	// 获取网络扫描请求参数
	req := request.GetNetScanInMessage()
	fmt.Printf("网络扫描 %v\n", req)
	t1 := time.Now()

	// 解析IP范围，生成待扫描的IP列表
	ipList, err := ip.ParseIPRange(req.IpRange)
	if err != nil {
		fmt.Println(err)
		// 解析失败时返回错误响应
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

	// 构建过滤IP集合，用于跳过指定IP的扫描
	filterIPList := map[string]struct{}{}
	for _, s := range req.FilterIPList {
		filterIPList[s] = struct{}{}
	}

	iface := req.Network          // 扫描使用的网络接口
	concurrency := 200            // 最大并发扫描数
	totalIPs := len(ipList)       // 待扫描IP总数
	processed := 0                // 已处理IP数量
	var processedMutex sync.Mutex // 保护processed变量的互斥锁

	// 创建并发控制信号量通道
	semaphore := make(chan struct{}, concurrency)

	fmt.Printf("开始扫描 %d 个IP地址，并发数: %d\n", totalIPs, concurrency)

	var wg sync.WaitGroup // 等待所有扫描协程完成
	for _, s := range ipList {
		// 跳过过滤列表中的IP
		if _, exists := filterIPList[s]; exists {
			continue
		}

		wg.Add(1)

		// 获取信号量，控制并发数
		semaphore <- struct{}{}

		// 启动协程执行ARP扫描
		go func(s string) {
			defer wg.Done()
			defer func() {
				// 协程结束时释放信号量
				<-semaphore
			}()

			// 通过指定网络接口执行ARP ping
			mac, _, err := arping.PingOverIfaceByName(net.ParseIP(s), iface)

			// 更新扫描进度
			processedMutex.Lock()
			processed++
			progress := float64(processed) / float64(totalIPs) * 100
			processedMutex.Unlock()

			// 目标主机不可达时跳过
			if err != nil {
				return
			}

			// 查询MAC地址对应的厂商信息
			manuf, _ := core.ManufQuery(mac.String())
			fmt.Printf("%s %s %s %.2f\n", s, mac, manuf, progress)

			// 发送扫描结果响应
			nc.cmdResponseChan <- &node_rpc.CmdResponse{
				CmdType: node_rpc.CmdType_cmdNetScanType,
				TaskID:  request.TaskID,
				NodeID:  nc.config.System.Uid,
				NetScanOutMessage: &node_rpc.NetScanOutMessage{
					End:      false,
					Progress: float32(progress),
					NetID:    req.NetID,
					Ip:       s,
					Mac:      mac.String(),
					Manuf:    manuf,
				},
			}
		}(s)
	}

	// 等待所有扫描协程完成
	wg.Wait()
	close(semaphore) // 关闭信号量通道

	// 发送扫描完成响应
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
	fmt.Printf("\n扫描完成，耗时: %v\n", time.Since(t1))
}
