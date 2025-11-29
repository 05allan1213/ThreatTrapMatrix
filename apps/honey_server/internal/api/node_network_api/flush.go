package node_network_api

// File: node_network_api.go
// Description: 节点网络API接口层，处理网络相关的HTTP请求与RPC交互

import (
	"context"
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/utils/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FlushView 处理节点网络视图刷新请求，通过RPC调用获取最新网络信息
func (NodeNetworkApi) FlushView(c *gin.Context) {
	cr := middleware.GetBind[models.IDRequest](c)

	var model models.NodeModel
	if err := global.DB.Take(&model, cr.Id).Error; err != nil {
		response.FailWithMsg("节点不存在", c)
		return
	}

	if model.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 使用封装的获取节点函数
	cmd, ok := grpc_service.GetNodeCommand(model.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 创建请求
	req := &node_rpc.CmdRequest{
		CmdType: node_rpc.CmdType_cmdNetworkFlushType,
		TaskID:  fmt.Sprintf("flush-%d", time.Now().UnixNano()),
		NetworkFlushInMessage: &node_rpc.NetworkFlushInMessage{
			FilterNetworkName: []string{"hy-"}, // 过滤名称以"hy-"结尾的网卡
		},
	}

	// 使用带超时的上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 发送请求
	select {
	case cmd.ReqChan <- req:
		logrus.Debugf("Sent flush request to node %s", model.Uid)
	case <-ctx.Done():
		response.FailWithMsg("发送命令超时", c)
		return
	}

	// 等待响应
	select {
	case res := <-cmd.ResChan:
		logrus.Debugf("Received flush response from node %s", model.Uid)
		response.OkWithData(res.NetworkFlushOutMessage, c)
	case <-ctx.Done():
		response.FailWithMsg("获取响应超时", c)
		return
	}
}
