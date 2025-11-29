package node_network_api

// File: node_network_api.go
// Description: 节点网络API接口层，处理网络相关的HTTP请求与RPC交互

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// FlushView 处理节点网络视图刷新请求，通过RPC调用获取最新网络信息
func (NodeNetworkApi) FlushView(c *gin.Context) {
	cr := middleware.GetBind[models.IDRequest](c)
	var model models.NodeModel
	err := global.DB.Take(&model, cr.Id).Error
	if err != nil {
		response.FailWithMsg("节点不存在", c)
		return
	}

	if model.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 判断节点在不在
	_, ok := grpc_service.NodeCommandMap[model.Uid]
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 构建网络刷新RPC请求并发送至通道
	grpc_service.NodeCommandMap[model.Uid].ReqChan <- &node_rpc.CmdRequest{
		CmdType: node_rpc.CmdType_cmdNetworkFlushType,
		TaskID:  "xx",
		NetworkFlushInMessage: &node_rpc.NetworkFlushInMessage{
			FilterNetworkName: []string{"hy-"}, // 过滤名称以"hy-"结尾的网卡
		},
	}

	// 阻塞等待RPC响应结果
	res := <-grpc_service.NodeCommandMap[model.Uid].ResChan
	fmt.Println("网卡刷新数据", res) // 日志打印刷新结果（调试用）

	// 返回网络刷新结果给客户端
	response.OkWithData(res.NetworkFlushOutMessage, c)
}
