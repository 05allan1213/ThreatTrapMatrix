package cron_service

// File: honey_node/service/cron_service/resource.go
// Description: 定时资源上报任务实现，负责采集节点资源信息并通过gRPC上报至管理端，包含完整的错误处理与日志记录

import (
	"context"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/info"
	"os"

	"github.com/sirupsen/logrus"
)

// Resource 节点资源采集与上报核心函数
func Resource() {
	// 前置检查：确保gRPC客户端已初始化并连接，避免无效上报
	if global.GrpcClient == nil {
		logrus.Errorf("管理端未连接，放弃上报")
		return
	}

	// 获取当前工作目录作为节点数据存储路径（用于统计该路径的磁盘占用）
	nodePath, _ := os.Getwd()

	// 采集系统资源信息（CPU、内存、磁盘等）
	resourceInfo, err := info.GetResourceInfo(nodePath)
	if err != nil {
		logrus.Errorf("节点资源信息获取失败 %s", err)
		return // 采集失败则终止本次上报
	}

	// 构造gRPC上报请求（映射资源信息到protobuf结构体）
	req := node_rpc.NodeResourceRequest{
		NodeUid: global.Config.System.Uid, // 节点唯一标识，用于管理端识别节点
		ResourceInfo: &node_rpc.ResourceMessage{
			CpuCount:              resourceInfo.CpuCount,
			CpuUseRate:            resourceInfo.CpuUseRate,
			MemTotal:              resourceInfo.MemTotal,
			MemUseRate:            resourceInfo.MemUseRate,
			DiskTotal:             resourceInfo.DiskTotal,
			DiskUseRate:           resourceInfo.DiskUseRate,
			NodePath:              resourceInfo.NodePath,
			NodeResourceOccupancy: resourceInfo.NodeResourceOccupancy,
		},
	}

	// 调用gRPC接口上报资源信息至管理端
	_, err = global.GrpcClient.NodeResource(context.Background(), &req)
	if err != nil {
		logrus.Errorf("节点资源信息上报失败 %s", err)
		return // 上报失败记录日志，不中断后续任务
	}

	// 上报成功记录日志，便于监控任务执行状态
	logrus.Infof("节点资源信息上报成功 %v", req)
}
