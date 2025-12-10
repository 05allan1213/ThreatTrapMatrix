package grpc_service

// File: honey_server/service/grpc_service/status_create_ip.go
// Description: 节点gRPC服务实现，处理诱捕IP创建状态的上报请求并更新数据库状态

import (
	"context"
	"fmt"
	"honey_server/internal/core"
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"
	"honey_server/internal/service/mq_service"
	"honey_server/internal/service/redis_service/net_lock"

	"gorm.io/gorm"
)

// StatusCreateIP 处理节点上报的诱捕IP创建状态请求
func (NodeService) StatusCreateIP(ctx context.Context, request *node_rpc.StatusCreateIPRequest) (pd *node_rpc.BaseResponse, err error) {
	pd = new(node_rpc.BaseResponse) // 初始化gRPC响应对象
	log := core.GetLogger().WithField("logID", request.LogID)
	log.WithField("request_data", request).Infof("接收创建ip回调")
	// 查询对应的诱捕IP记录，验证记录存在性
	var honeyIPModel models.HoneyIpModel
	err1 := global.DB.Take(&honeyIPModel, request.HoneyIPID).Error
	if err1 != nil {
		return nil, fmt.Errorf("诱捕ip不存在 %d", request.HoneyIPID)
	}

	net_lock.UnLock(honeyIPModel.NetID)

	// 定义状态：2表示创建成功，3表示创建失败
	var status int8 = 2
	if request.ErrMsg != "" {
		status = 3 // 节点上报错误信息，标记为创建失败
		log.WithField("error", request.ErrMsg).Errorf("failed to create honey ip")
	}

	// 更新数据库中诱捕IP的状态、MAC地址、绑定的网络接口等信息
	global.DB.Model(&honeyIPModel).Updates(models.HoneyIpModel{
		Mac:      request.Mac,     // 节点上报的虚拟接口MAC地址
		Network:  request.Network, // 绑定的物理网络接口名称
		Status:   status,          // 创建结果状态（2成功/3失败）
		ErrorMsg: request.ErrMsg,  // 错误信息
	})

	mq_service.SendWsMsg(mq_service.WsMsgType{
		LogID:  request.LogID,
		Type:   1,
		NetID:  honeyIPModel.NetID,
		NodeID: honeyIPModel.NodeID,
	})

	var nodeModel models.NodeModel
	global.DB.Take(&nodeModel, honeyIPModel.NodeID)
	global.DB.Model(&nodeModel).Update("honey_ip_count", gorm.Expr("honey_ip_count + 1"))
	var netModel models.NetModel
	global.DB.Take(&netModel, honeyIPModel.NetID)
	global.DB.Model(&netModel).Update("honey_ip_count", gorm.Expr("honey_ip_count + 1"))

	return
}
