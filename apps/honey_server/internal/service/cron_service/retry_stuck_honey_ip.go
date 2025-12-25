package cron_service

// File: honey_server/service/cron_service/retry_stuck_honey_ip.go
// Description: 定时扫描并重试长时间停留在 "创建中" 状态的诱捕IP记录

import (
	"honey_server/internal/core"
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/service/mq_service"
	"time"
)

// RetryStuckHoneyIP 扫描并重试超时的诱捕IP创建任务
func RetryStuckHoneyIP() {
	log := core.GetLogger().WithField("task", "retry_stuck_honey_ip")
	log.Info("开始扫描超时的诱捕IP记录")

	const maxRetryCount = 3 // 最大重试次数

	// 查询Status=1且创建时间超过5分钟的记录
	var stuckIPs []models.HoneyIpModel
	timeout := time.Now().Add(-5 * time.Minute)

	err := global.DB.Preload("NetModel").Preload("NodeModel").
		Where("status = ? AND created_at < ?", 1, timeout).
		Find(&stuckIPs).Error

	if err != nil {
		log.WithField("error", err).Error("查询超时记录失败")
		return
	}

	if len(stuckIPs) == 0 {
		log.Info("未发现超时记录")
		return
	}

	log.WithField("count", len(stuckIPs)).Warn("发现超时记录，开始重试")

	// 重试每个超时记录
	for _, honeyIP := range stuckIPs {
		retryLog := log.WithField("honey_ip_id", honeyIP.ID).
			WithField("ip", honeyIP.IP).
			WithField("retry_count", honeyIP.RetryCount).
			WithField("created_at", honeyIP.CreatedAt)

		// 检查是否超过最大重试次数
		if honeyIP.RetryCount >= maxRetryCount {
			retryLog.Warn("超过最大重试次数，标记为失败")
			global.DB.Model(&honeyIP).Updates(models.HoneyIpModel{
				Status:   3,
				ErrorMsg: "超过最大重试次数",
			})
			continue
		}

		// 判断是否是探针IP
		isTan := honeyIP.NetModel.IP == honeyIP.IP

		// 重新发送MQ消息
		mq_service.SendCreateIPMsg(honeyIP.NodeModel.Uid, mq_service.CreateIPRequest{
			HoneyIPID: honeyIP.ID,
			IP:        honeyIP.IP,
			Mask:      honeyIP.NetModel.Mask,
			Network:   honeyIP.NetModel.Network,
			IsTan:     isTan,
			LogID:     "",
		})

		// 更新重试次数
		global.DB.Model(&honeyIP).Update("retry_count", honeyIP.RetryCount+1)

		retryLog.Info("已重新投递MQ消息")
	}

	log.WithField("retry_count", len(stuckIPs)).Info("重试任务完成")
}
