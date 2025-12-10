package command

// File: honey_node/service/command/command_node_remove.go
// Description: 节点命令处理模块，实现节点移除命令的核心逻辑，包含运行时数据清空、RPC响应返回及节点进程正常退出

import (
	"honey_node/internal/core"
	"honey_node/internal/flags"
	"honey_node/internal/rpc/node_rpc"
	"os"
)

// CmdNodeRemove 处理节点移除RPC命令
func (nc *NodeClient) CmdNodeRemove(request *node_rpc.CmdRequest) {
	// 清空节点运行时标识/状态数据
	log := core.GetLogger().WithField("logID", request.LogID)
	log.Infof("管理删除节点")
	flags.Clear(log)

	// 构造节点移除命令的响应数据，发送到RPC响应通道，告知服务端命令执行完成
	nc.cmdResponseChan <- &node_rpc.CmdResponse{
		CmdType:              node_rpc.CmdType_cmdNodeRemoveType, // 命令类型：节点移除
		TaskID:               request.TaskID,                     // 关联的任务ID，保持请求/响应上下文一致
		NodeID:               nc.config.System.Uid,               // 当前节点唯一标识，标识响应所属节点
		NodeRemoveOutMessage: &node_rpc.NodeRemoveOutMessage{},   // 节点移除响应体
	}

	// 正常终止节点进程
	os.Exit(0)
}
