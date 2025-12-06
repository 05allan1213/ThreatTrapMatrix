package mq_service

// File: matrix_server/service/mq_service/rev_batch_update_deploy_status_mq.go
// Description: 批量更新部署状态消息处理模块，实现更新部署状态回调的核心业务逻辑，包含更新进度统计、端口绑定失败状态更新及更新完成后的分布式锁释放

import (
	"matrix_server/internal/global"
	"matrix_server/internal/models"
	"matrix_server/internal/service/redis_service/net_lock"
	"matrix_server/internal/service/redis_service/net_progress"

	"github.com/sirupsen/logrus"
)

// UpdateDeployStatusRequest 批量更新部署状态回调的消息结构体
type UpdateDeployStatusRequest struct {
	NetID    uint         `json:"netID"`    // 子网ID，关联目标操作子网
	IP       string       `json:"ip"`       // 已执行更新部署操作的IP地址
	LogID    string       `json:"logID"`    // 日志ID，用于关联操作全链路日志
	ErrorMsg string       `json:"errorMsg"` // IP级更新部署执行失败时的错误信息
	PortList []PortStatus `json:"portList"` // 该IP下各端口的更新部署状态列表
}

// PortStatus 端口更新部署状态结构体
type PortStatus struct {
	Port     int    `json:"port"`     // 执行更新部署操作的端口号
	ErrorMsg string `json:"errorMsg"` // 端口级更新部署（绑定）失败时的错误信息
}

// revBatchUpdateDeployStatusMq 批量更新部署状态消息的具体处理函数
func revBatchUpdateDeployStatusMq(data UpdateDeployStatusRequest) {
	// 从Redis获取当前子网的更新部署进度信息
	netDeployInfo, err := net_progress.Get(data.NetID)
	if err != nil {
		logrus.Errorf("获取子网更新部署信息失败 %s，子网ID：%d", err, data.NetID)
		return
	}

	// 更新更新部署进度统计：已完成数+1，若IP级执行出错则记录错误IP及错误数
	netDeployInfo.CompletedCount++
	if data.ErrorMsg != "" {
		netDeployInfo.ErrorCount++
		netDeployInfo.ErrorIpList = append(netDeployInfo.ErrorIpList, net_progress.ErrorIp{
			Ip:  data.IP,
			Msg: data.ErrorMsg,
		})
	}

	// 打印子网更新部署进度日志（已完成数/总数 + 百分比）
	logrus.Infof("%d当前子网，正在更新部署%d个，共%d个 进度：%.2f%%", data.NetID,
		netDeployInfo.CompletedCount,
		netDeployInfo.AllCount,
		(float64(netDeployInfo.CompletedCount)/float64(netDeployInfo.AllCount))*100,
	)

	// 将更新后的更新部署进度信息保存到Redis
	err = net_progress.Set(data.NetID, netDeployInfo)
	if err != nil {
		logrus.Errorf("设置子网更新部署信息失败 %s，子网ID：%d", err, data.NetID)
		return
	}

	// 遍历端口状态列表，处理端口级绑定失败的情况
	for _, status := range data.PortList {
		if status.ErrorMsg != "" {
			// 记录端口绑定失败日志（IP+端口+错误信息）
			logrus.Errorf("端口绑定失败 %s %d %s", data.IP, status.Port, status.ErrorMsg)
			// 更新端口状态为绑定失败（状态2）
			var portModel models.HoneyPortModel
			global.DB.Take(&portModel, "net_id = ? and ip = ? and port = ?", data.NetID, data.IP, status.Port).Update("status", 2)
		}
	}

	// 判定子网更新部署完成：已完成数等于总数时释放分布式锁
	if netDeployInfo.CompletedCount == netDeployInfo.AllCount {
		// 释放子网分布式锁，允许后续操作
		_, _ = net_lock.UnLock(data.NetID)
		logrus.Infof("子网%d更新部署完成 解锁", data.NetID)
	}
}
