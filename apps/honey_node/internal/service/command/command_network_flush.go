package command

// File: honey_node/service/command/command_network_flush.go
// Description: 节点客户端网卡刷新命令处理逻辑，实现网卡信息收集及响应构建

import (
	"honey_node/internal/rpc/node_rpc"

	"github.com/sirupsen/logrus"
)

// CmdNetworkFlush 处理服务端下发的网卡刷新命令，收集节点网卡信息并返回响应
func (nc *NodeClient) CmdNetworkFlush(request *node_rpc.CmdRequest) {
	logrus.Info("处理网卡刷新命令")

	// 解析请求中的网卡过滤条件
	var filters []string
	if request.NetworkFlushInMessage != nil && len(request.NetworkFlushInMessage.FilterNetworkName) > 0 {
		filters = request.NetworkFlushInMessage.FilterNetworkName
	}

	// 根据过滤条件获取节点网卡列表
	networkList, err := nc.getNetworkList(filters)
	if err != nil {
		logrus.Errorf("获取网络列表失败: %v", err)
		return
	}

	// 构建网卡刷新响应数据
	response := &node_rpc.CmdResponse{
		CmdType: node_rpc.CmdType_cmdNetworkFlushType,
		TaskID:  request.TaskID,
		NodeID:  nc.config.System.Uid,
		NetworkFlushOutMessage: &node_rpc.NetworkFlushOutMessage{
			NetworkList: networkList,
		},
	}

	// 将响应发送至通道，等待传输至服务端
	select {
	case nc.cmdResponseChan <- response:
		logrus.Debugf("已将响应加入发送队列: %+v", response)
	case <-nc.ctx.Done():
		logrus.Warn("上下文已取消，丢弃响应")
	}
}
