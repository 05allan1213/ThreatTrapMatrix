package grpc_service

// File: honey_server/service/grpc_service/register.go
// Description: 节点注册gRPC服务接口实现，处理节点注册请求，包含节点创建及状态更新逻辑

import (
	"context"
	"errors"
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"

	"github.com/sirupsen/logrus"
)

// Register 节点注册接口实现
func (NodeService) Register(ctx context.Context, request *node_rpc.RegisterRequest) (pd *node_rpc.BaseResponse, err error) {
	pd = new(node_rpc.BaseResponse) // 初始化gRPC响应结构体

	// 获取节点唯一标识（用于判断节点是否已存在）
	uid := request.NodeUid
	var model models.NodeModel

	// 根据节点UID查询数据库中是否已存在该节点
	err1 := global.DB.Take(&model, "uid = ?", uid).Error
	if err1 != nil {
		// 节点不存在，创建新节点记录
		model = models.NodeModel{
			Title:  request.SystemInfo.HostName, // 节点名称
			Uid:    uid,                         // 节点唯一标识
			IP:     request.Ip,                  // 节点ip
			Mac:    request.Mac,                 // 节点mac
			Status: 1,                           // 节点状态：1-在线
			SystemInfo: models.NodeSystemInfo{ // 节点系统信息
				NodeVersion:         request.Version,
				NodeCommit:          request.Commit,
				HostName:            request.SystemInfo.HostName,
				DistributionVersion: request.SystemInfo.DistributionVersion,
				CoreVersion:         request.SystemInfo.CoreVersion,
				SystemType:          request.SystemInfo.SystemType,
				StartTime:           request.SystemInfo.StartTime,
			},
		}
		// 执行节点记录创建操作
		err1 = global.DB.Create(&model).Error
		if err1 != nil {
			logrus.Errorf("节点创建失败 %s", err1) // 记录节点创建失败日志
			return nil, errors.New("节点创建失败") // 返回错误信息
		}
	}

	// 处理网卡记录 - 先查询再决定操作
	for _, message := range request.NetworkList {
		var existingNetwork models.NodeNetworkModel
		// 查询是否已存在相同的网卡记录（同一节点下相同网卡名称）
		err = global.DB.Where("node_id = ? AND network = ?", model.ID, message.Network).First(&existingNetwork).Error
		
		if err != nil {
			// 记录不存在，创建新记录
			networkRecord := models.NodeNetworkModel{
				NodeID:  model.ID,
				Network: message.Network,
				IP:      message.Ip,
				Mask:    int8(message.Mask),
				Status:  2,
			}
			err = global.DB.Create(&networkRecord).Error
			if err != nil {
				logrus.Errorf("节点网卡保存失败 %s", err)
				return nil, errors.New("节点网卡保存失败")
			}
		} else {
			// 记录已存在，检查是否需要更新
			if existingNetwork.IP != message.Ip || existingNetwork.Mask != int8(message.Mask) {
				// IP或掩码有变化，更新记录
				updates := map[string]interface{}{
					"ip":   message.Ip,
					"mask": int8(message.Mask),
				}
				err = global.DB.Model(&existingNetwork).Updates(updates).Error
				if err != nil {
					logrus.Errorf("节点网卡更新失败 %s", err)
					return nil, errors.New("节点网卡更新失败")
				}
			}
		}
	}

	// 节点已存在，检查状态是否为在线，非在线则更新为在线
	if model.Status != 1 {
		global.DB.Model(&model).Update("status", 1)
	}

	return
}
