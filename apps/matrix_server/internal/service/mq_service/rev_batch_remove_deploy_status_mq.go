package mq_service

// File: matrix_server/service/mq_service/rev_batch_remove_deploy_status_mq.go
// Description: 批量删除部署状态消息处理模块，实现删除部署状态回调的核心业务逻辑，包含删除进度更新、诱捕IP关联端口清理、IP记录删除及删除完成后的分布式锁释放

import (
	"matrix_server/internal/global"
	"matrix_server/internal/models"
	"matrix_server/internal/service/redis_service/net_lock"
	"matrix_server/internal/service/redis_service/net_progress"

	"github.com/sirupsen/logrus"
)

// RemoveDeployStatusRequest 批量删除部署状态回调的消息结构体
type RemoveDeployStatusRequest struct {
	NetID    uint   `json:"netID"`    // 子网ID，关联目标操作子网
	IP       string `json:"ip"`       // 已执行删除部署操作的IP地址
	LogID    string `json:"logID"`    // 日志ID，用于关联操作全链路日志
	ErrorMsg string `json:"errorMsg"` // 删除部署执行失败时的错误信息
}

// revBatchRemoveDeployStatusMq 批量删除部署状态消息的具体处理函数
func revBatchRemoveDeployStatusMq(data RemoveDeployStatusRequest) {
	// 从Redis获取当前子网的删除部署进度信息
	netDeployInfo, err := net_progress.Get(data.NetID)
	if err != nil {
		logrus.Errorf("获取子网删除部署信息失败 %s，子网ID：%d", err, data.NetID)
		return
	}

	// 更新删除部署进度统计：已完成数+1，若有错误则记录错误IP及错误数
	netDeployInfo.CompletedCount++
	if data.ErrorMsg != "" {
		netDeployInfo.ErrorCount++
		netDeployInfo.ErrorIpList = append(netDeployInfo.ErrorIpList, net_progress.ErrorIp{
			Ip:  data.IP,
			Msg: data.ErrorMsg,
		})
	}

	// 打印子网删除部署进度日志（已完成数/总数 + 百分比）
	logrus.Infof("%d当前子网，正在删除部署%d个，共%d个 进度：%.2f%%", data.NetID,
		netDeployInfo.CompletedCount,
		netDeployInfo.AllCount,
		(float64(netDeployInfo.CompletedCount)/float64(netDeployInfo.AllCount))*100,
	)

	// 将更新后的删除部署进度信息保存到Redis
	err = net_progress.Set(data.NetID, netDeployInfo)
	if err != nil {
		logrus.Errorf("设置子网删除部署信息失败 %s，子网ID：%d", err, data.NetID)
		return
	}

	// 每20个推送一次
	if netDeployInfo.CompletedCount%20 == 0 {
		SendWsMsg(WsMsgType{
			Type:  1,
			NetID: data.NetID,
		})
	}

	// 查询当前IP对应的诱捕IP记录（预加载关联的端口列表）
	var model models.HoneyIpModel
	err = global.DB.Preload("PortList").Take(&model, "net_id = ? and ip = ?", data.NetID, data.IP).Error
	if err == nil {
		// 清理诱捕IP关联的端口记录
		if len(model.PortList) > 0 {
			global.DB.Delete(&model.PortList)
			logrus.Infof("删除IP[%s]关联端口 %d 个", data.IP, len(model.PortList))
		}
		// 删除诱捕IP记录
		global.DB.Delete(&model)
		logrus.Infof("删除子网[%d]下诱捕IP：%s", data.NetID, data.IP)
	}

	// 判定子网删除部署完成：已完成数等于总数时释放分布式锁
	if netDeployInfo.CompletedCount == netDeployInfo.AllCount {
		// 释放子网分布式锁，允许后续操作
		_, _ = net_lock.UnLock(data.NetID)
		logrus.Infof("子网%d删除部署完成 解锁", data.NetID)
		SendWsMsg(WsMsgType{
			Type:  1,
			NetID: data.NetID,
		})
	}
}
