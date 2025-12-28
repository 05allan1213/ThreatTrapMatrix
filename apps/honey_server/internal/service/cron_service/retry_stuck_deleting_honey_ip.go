package cron_service

// File: honey_server/service/cron_service/retry_stuck_deleting_honey_ip.go
// Description: 定时扫描并重试长时间停留在 "删除中" 状态的诱捕IP记录

import (
	"honey_server/internal/core"
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/service/mq_service"
	"time"
)

// RetryStuckDeletingHoneyIP 扫描并重试超时的诱捕IP删除任务
func RetryStuckDeletingHoneyIP() {
	log := core.GetLogger().WithField("task", "retry_stuck_deleting_honey_ip")
	log.Info("开始扫描超时的删除中诱捕IP记录")

	const maxRetryCount = 3 // 最大重试次数

	// 查询Status=4且更新时间超过5分钟的记录
	var stuckIPs []models.HoneyIpModel
	timeout := time.Now().Add(-5 * time.Minute)

	err := global.DB.Preload("NetModel").Preload("NodeModel").
		Where("status = ? AND updated_at < ?", 4, timeout).
		Find(&stuckIPs).Error

	if err != nil {
		log.WithField("error", err).Error("查询超时删除记录失败")
		return
	}

	if len(stuckIPs) == 0 {
		log.Info("未发现超时删除记录")
		return
	}

	log.WithField("count", len(stuckIPs)).Warn("发现超时删除记录，开始重试")

	// 按NetID分组，批量重试
	netGroups := make(map[uint][]uint)
	ipRetryMap := make(map[uint]models.HoneyIpModel)

	for _, honeyIP := range stuckIPs {
		netGroups[honeyIP.NetID] = append(netGroups[honeyIP.NetID], honeyIP.ID)
		ipRetryMap[honeyIP.ID] = honeyIP
	}

	// 按网络分组重试
	for netID, ipIDs := range netGroups {
		retryLog := log.WithField("net_id", netID).
			WithField("honey_ip_ids", ipIDs).
			WithField("count", len(ipIDs))

		// 检查第一个IP的重试次数（同一批次应该一致）
		firstIP := ipRetryMap[ipIDs[0]]

		if firstIP.RetryCount >= maxRetryCount {
			retryLog.Warn("超过最大重试次数，标记为失败")
			// 批量标记为失败
			global.DB.Model(&models.HoneyIpModel{}).
				Where("id IN ?", ipIDs).
				Updates(models.HoneyIpModel{
					Status:   3,
					ErrorMsg: "删除超时，重试耗尽",
				})
			continue
		}

		// 重建IpList（包含完整的删除信息）
		var ipList []mq_service.IpInfo
		for _, ipID := range ipIDs {
			honeyIP := ipRetryMap[ipID]
			isTan := honeyIP.NetModel.IP == honeyIP.IP
			ipList = append(ipList, mq_service.IpInfo{
				HoneyIPID: honeyIP.ID,
				IP:        honeyIP.IP,
				Network:   honeyIP.Network,
				IsTan:     isTan,
			})
		}

		// 重新发送删除MQ消息
		mq_service.SendDeleteIPMsg(firstIP.NodeModel.Uid, mq_service.DeleteIPRequest{
			IpList: ipList,
			NetID:  netID,
			LogID:  "",
		})

		// 更新重试次数
		global.DB.Model(&models.HoneyIpModel{}).
			Where("id IN ?", ipIDs).
			Update("retry_count", firstIP.RetryCount+1)

		retryLog.Info("已重新投递删除MQ消息")
	}

	log.WithField("retry_count", len(stuckIPs)).Info("删除重试任务完成")
}
