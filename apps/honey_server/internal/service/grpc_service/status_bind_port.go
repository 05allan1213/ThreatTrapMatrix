package grpc_service

// File: honey_server/grpc_service/status_bind_port.go
// Description: 节点gRPC服务处理模块，实现端口绑定状态回调的RPC接口逻辑，接收节点上报的端口绑定状态，更新诱捕IP关联端口的状态信息

import (
	"context"
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"

	"github.com/sirupsen/logrus"
)

// StatusBindPort 节点端口绑定状态回调的gRPC接口实现
func (NodeService) StatusBindPort(ctx context.Context, request *node_rpc.StatusBindPortRequest) (pd *node_rpc.BaseResponse, err error) {
	// 初始化gRPC响应结构体
	pd = new(node_rpc.BaseResponse)

	// 查询指定ID的诱捕IP记录（预加载关联的端口列表）
	var honeyIPModel models.HoneyIpModel
	logrus.Infof("端口转发回调 %d %#v", request.HoneyIPID, request.PortInfoList)
	err1 := global.DB.Preload("PortList").Take(&honeyIPModel, request.HoneyIPID).Error
	if err1 != nil {
		// 诱捕IP不存在时返回错误
		return nil, fmt.Errorf("诱捕ip不存在 %d", request.HoneyIPID)
	}

	// 构建端口号到端口模型的映射
	var portMap = map[int64]*models.HoneyPortModel{}
	for _, model := range honeyIPModel.PortList {
		portMap[int64(model.Port)] = &model
	}

	// 遍历节点上报的端口状态列表，仅处理携带错误信息的端口（绑定失败）
	for _, i2 := range request.PortInfoList {
		if i2.Msg != "" {
			// 根据端口号查找本地端口模型
			model, ok := portMap[i2.Port]
			if !ok {
				// 端口信息不存在时记录错误日志，跳过当前端口处理
				logrus.Errorf("端口信息不存在 %d", i2.Port)
				continue
			}
			// 更新端口状态为节点上报的错误信息
			global.DB.Model(model).Update("status", i2.Msg)
		}
	}

	// 无错误时返回初始化的响应结构体和nil错误
	return
}
