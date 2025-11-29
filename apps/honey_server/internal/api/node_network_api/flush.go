package node_network_api

// File: honey_server/api/node_network_api/flush.go
// Description: 节点网卡刷新API接口

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// FlushView 处理节点网络视图刷新请求，向指定节点发送网卡刷新命令并返回结果
func (NodeNetworkApi) FlushView(c *gin.Context) {
	// 获取请求绑定的节点ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	// 查询节点信息
	var model models.NodeModel
	if err := global.DB.Take(&model, cr.Id).Error; err != nil {
		response.FailWithMsg("节点不存在", c)
		return
	}

	// 检查节点运行状态
	if model.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 获取节点的命令交互实例（验证节点是否在线）
	cmd, ok := grpc_service.GetNodeCommand(model.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 构建网卡刷新命令请求
	req := &node_rpc.CmdRequest{
		CmdType: node_rpc.CmdType_cmdNetworkFlushType,
		TaskID:  fmt.Sprintf("flush-%d", time.Now().UnixNano()), // 生成唯一任务ID
		NetworkFlushInMessage: &node_rpc.NetworkFlushInMessage{
			FilterNetworkName: []string{"hy-"}, // 过滤hy-后缀的网卡
		},
	}

	// 创建带超时的上下文（30秒超时）
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 发送命令请求至节点
	select {
	case cmd.ReqChan <- req:
		logrus.Debugf("Sent flush request to node %s", model.Uid)
	case <-ctx.Done():
		response.FailWithMsg("发送命令超时", c)
		return
	}

	// 等待节点响应结果
	select {
	case res := <-cmd.ResChan:
		logrus.Debugf("Received flush response from node %s", model.Uid)
		response.OkWithData(res.NetworkFlushOutMessage, c)
	case <-ctx.Done():
		response.FailWithMsg("获取响应超时", c)
		return
	}
}
