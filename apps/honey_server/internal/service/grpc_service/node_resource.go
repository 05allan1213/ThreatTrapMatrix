package grpc_service

// File: honey_server/service/grpc_service/node_resource.go
// Description: 节点资源信息更新gRPC服务接口实现，接收节点上报的资源数据并更新到数据库

import (
	"context"
	"errors"
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"

	"github.com/sirupsen/logrus"
)

// NodeResource 节点资源信息更新接口实现
func (NodeService) NodeResource(ctx context.Context, request *node_rpc.NodeResourceRequest) (pd *node_rpc.BaseResponse, err error) {
	pd = new(node_rpc.BaseResponse) // 初始化gRPC响应结构体

	// 获取节点唯一标识，用于查询节点是否存在
	uid := request.NodeUid
	var model models.NodeModel

	// 根据节点UID查询数据库，验证节点是否已注册
	err1 := global.DB.Take(&model, "uid = ?", uid).Error
	if err1 != nil {
		return nil, errors.New("节点不存在") // 节点未注册则返回错误
	}

	// 组装节点资源信息更新数据
	newModel := models.NodeModel{
		Resource: models.NodeResource{
			CpuCount:              int(request.ResourceInfo.CpuCount),         // CPU内核数
			CpuUseRate:            float64(request.ResourceInfo.CpuUseRate),   // CPU使用率
			MemTotal:              request.ResourceInfo.MemTotal,              // 内存容量
			MemUseRate:            float64(request.ResourceInfo.MemUseRate),   // 内存使用率
			DiskTotal:             request.ResourceInfo.DiskTotal,             // 磁盘容量
			DiskUseRate:           float64(request.ResourceInfo.DiskUseRate),  // 磁盘使用率
			NodePath:              request.ResourceInfo.NodePath,              // 节点部署目录
			NodeResourceOccupancy: request.ResourceInfo.NodeResourceOccupancy, // 节点部署目录资源占用
		},
	}

	// 执行节点资源信息更新操作
	err1 = global.DB.Model(&model).Updates(newModel).Error
	if err1 != nil {
		logrus.Errorf("节点资源状态更新失败 %s", err1) // 记录资源更新失败日志
		return nil, errors.New("节点资源状态更新失败") // 返回更新失败错误
	}
	return
}
