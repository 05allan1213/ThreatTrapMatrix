package command

// File: honey_node/service/command/command_net_scan.go
// Description: 节点客户端网络扫描命令处理逻辑，实现网络扫描任务执行及响应反馈机制

import (
	"fmt"
	"honey_node/internal/rpc/node_rpc"
	"time"
)

// CmdNetScan 处理服务端下发的网络扫描命令，执行扫描任务并分阶段返回响应
func (nc *NodeClient) CmdNetScan(request *node_rpc.CmdRequest) {
	// 提取网络扫描请求参数
	req := request.GetNetScanInMessage()
	fmt.Printf("开始执行网络扫描任务: %v\n", req)

	// 发送扫描进度中间响应（模拟扫描过程）
	nc.cmdResponseChan <- &node_rpc.CmdResponse{
		CmdType: node_rpc.CmdType_cmdNetScanType,
		TaskID:  request.TaskID,
		NodeID:  nc.config.System.Uid,
		NetScanOutMessage: &node_rpc.NetScanOutMessage{
			End:      false,
			Progress: 0,
			Ip:       "192.168.100.1", // 示例扫描IP
		},
	}

	// 模拟扫描耗时操作
	time.Sleep(2 * time.Second)

	// 发送扫描完成最终响应
	nc.cmdResponseChan <- &node_rpc.CmdResponse{
		CmdType: node_rpc.CmdType_cmdNetScanType,
		TaskID:  request.TaskID,
		NodeID:  nc.config.System.Uid,
		NetScanOutMessage: &node_rpc.NetScanOutMessage{
			End:      true,
			Progress: 100,
		},
	}
}
