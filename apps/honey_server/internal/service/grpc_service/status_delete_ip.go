package grpc_service

// File: honey_server/service/grpc_service/status_delete_ip.go
// Description: 节点gRPC服务实现，处理诱捕IP删除状态的上报请求并执行数据库删除操作

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

// StatusDeleteIP 处理节点上报的诱捕IP删除状态请求
func (NodeService) StatusDeleteIP(ctx context.Context, request *node_rpc.StatusDeleteIPRequest) (pd *node_rpc.BaseResponse, err error) {
	pd = new(node_rpc.BaseResponse) // 初始化gRPC响应对象
	log := core.GetLogger().WithField("logID", request.LogID)
	log.WithField("request_data", request).Infof("接收批量删除ip回调")
	// 根据节点上报的ID列表查询对应的诱捕IP记录
	var honeyIPList []models.HoneyIpModel
	net_lock.UnLock(uint(request.NetID))
	global.DB.Find(&honeyIPList, "id in ?", request.HoneyIPIDList)

	// 校验查询结果：若没有找到任何记录，返回错误
	if len(honeyIPList) == 0 {
		return nil, fmt.Errorf("诱捕ip不存在 ")
	}

	// 执行数据库批量删除操作（删除查询到的诱捕IP记录）
	global.DB.Delete(&honeyIPList)

	firstHoneyIp := honeyIPList[0]

	var nodeModel models.NodeModel
	global.DB.Take(&nodeModel, firstHoneyIp.NodeID)
	global.DB.Model(&nodeModel).Update("honey_ip_count", gorm.Expr("honey_ip_count - ?", len(honeyIPList)))
	var netModel models.NetModel
	global.DB.Take(&netModel, firstHoneyIp.NetID)
	global.DB.Model(&netModel).Update("honey_ip_count", gorm.Expr("honey_ip_count - ?", len(honeyIPList)))

	mq_service.SendWsMsg(mq_service.WsMsgType{
		LogID:  request.LogID,
		Type:   1,
		NetID:  honeyIPList[0].NetID,
		NodeID: honeyIPList[0].NodeID,
	})
	return
}
