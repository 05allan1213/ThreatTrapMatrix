package mq_service

// File: matrix_server/service/mq_service/rev_batch_deploy_status_mq.go
// Description: 批量部署状态消息处理模块，实现部署状态回调的核心业务逻辑，包含部署进度更新、存活主机入库、诱捕IP状态更新及部署完成后的分布式锁释放

import (
	"matrix_server/internal/global"
	"matrix_server/internal/models"
	"matrix_server/internal/service/redis_service/net_lock"
	"matrix_server/internal/service/redis_service/net_progress"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// DeployStatusRequest 批量部署状态回调的消息结构体
type DeployStatusRequest struct {
	NetID    uint   `json:"netID"`    // 子网ID，关联目标操作子网
	IP       string `json:"ip"`       // 已执行部署操作的IP地址
	Mac      string `json:"mac"`      // IP对应的MAC地址
	LinkName string `json:"linkName"` // 网络接口名称
	LogID    string `json:"logID"`    // 日志ID，用于关联操作全链路日志
	ErrorMsg string `json:"errorMsg"` // 部署执行失败时的错误信息
	Manuf    string `json:"manuf"`    // 设备厂商名称
}

// revBatchDeployStatusMq 批量部署状态消息的具体处理函数
func revBatchDeployStatusMq(data DeployStatusRequest) {
	// 从Redis获取当前子网的部署进度信息
	netDeployInfo, err := net_progress.Get(data.NetID)
	if err != nil {
		logrus.Errorf("获取子网部署信息失败 %s", err)
		return
	}

	// 更新部署进度统计：已完成数+1，若有错误则记录错误IP及错误数
	// 节点补回调场景可能重复上报同一个IP，这里按IP去重，避免进度重复记账
	if netDeployInfo.HandledIPMap == nil {
		netDeployInfo.HandledIPMap = make(map[string]bool)
	}
	if netDeployInfo.HandledIPMap[data.IP] {
		logrus.Warnf("子网%d部署回调重复记账，忽略IP[%s]", data.NetID, data.IP)
		return
	}
	netDeployInfo.HandledIPMap[data.IP] = true
	netDeployInfo.CompletedCount++
	if data.ErrorMsg != "" {
		netDeployInfo.ErrorCount++
		netDeployInfo.ErrorIpList = append(netDeployInfo.ErrorIpList, net_progress.ErrorIp{
			Ip:  data.IP,
			Msg: data.ErrorMsg,
		})
	}

	// 打印子网部署进度日志（已完成数/总数 + 百分比）
	logrus.Infof("%d当前子网，正在部署%d个，共%d个，进度：%.2f%%", data.NetID,
		netDeployInfo.CompletedCount,
		netDeployInfo.AllCount,
		(float64(netDeployInfo.CompletedCount)/float64(netDeployInfo.AllCount))*100,
	)

	// 将更新后的部署进度信息保存到Redis
	err = net_progress.Set(data.NetID, netDeployInfo)
	if err != nil {
		logrus.Errorf("设置子网部署信息失败 %s", err)
		return
	}

	// 判定当前回调记账后是否达到最终完成条件，统一由尾部defer负责解锁
	shouldUnlock := netDeployInfo.CompletedCount == netDeployInfo.AllCount
	// 记录最终完成通知使用的节点ID，避免中途return导致解锁通知缺少节点信息
	var completionNodeID uint
	if shouldUnlock {
		defer func() {
			// 统一在函数返回前执行解锁，确保成功/失败/早退分支都能走到最终解锁逻辑
			ok, unlockErr := net_lock.UnLock(data.NetID)
			if unlockErr != nil {
				logrus.Errorf("子网%d部署完成解锁失败: %v", data.NetID, unlockErr)
			} else if !ok {
				logrus.Warnf("子网%d部署完成解锁未生效", data.NetID)
			} else {
				logrus.Infof("子网%d部署完成 解锁", data.NetID)
			}

			// 推送最终完成通知，允许前端刷新该子网的最终状态
			SendWsMsg(WsMsgType{
				LogID:  data.LogID,
				Type:   1,
				NetID:  data.NetID,
				NodeID: completionNodeID,
			})
		}()
	}

	// 每20个推送一次
	if netDeployInfo.CompletedCount%20 == 0 {
		SendWsMsg(WsMsgType{
			LogID: data.LogID,
			Type:  1,
			NetID: data.NetID,
		})
	}

	// 查询当前IP对应的诱捕IP记录
	var honeyIP models.HoneyIpModel
	err = global.DB.Take(&honeyIP, "net_id = ? and ip = ?", data.NetID, data.IP).Error
	if err != nil {
		logrus.Errorf("honeyIP记录不存在，IP：%s 子网ID：%d", data.IP, data.NetID)
		return
	}
	// 缓存节点ID，供尾部统一解锁通知使用
	completionNodeID = honeyIP.NodeID

	// 处理存活主机场景：将存活主机信息入库，并删除对应的诱捕IP记录
	if data.ErrorMsg == "存活主机" {
		global.DB.Create(&models.HostModel{
			NodeID: honeyIP.NodeID,
			NetID:  data.NetID,
			IP:     data.IP,
			Mac:    data.Mac,
			Manuf:  data.Manuf,
		})
		global.DB.Delete(&honeyIP)
		return
	}

	// 组装诱捕IP更新数据：更新MAC、接口名、错误信息及部署状态
	var hp = models.HoneyIpModel{
		Mac:      data.Mac,
		Network:  data.LinkName,
		ErrorMsg: data.ErrorMsg,
		Status:   2, // 状态2：部署成功
	}
	// 部署失败时更新状态为3（部署失败）
	if data.ErrorMsg != "" {
		hp.Status = 3
	}

	// 更新诱捕IP记录
	err = global.DB.Model(&honeyIP).Updates(hp).Error
	if err != nil {
		logrus.Errorf("记录更新失败 %s，IP：%s 子网ID：%d", err, data.IP, data.NetID)
	}

	// 子网部署全部完成后，更新节点与子网下的诱捕IP统计数量
	if shouldUnlock {
		var nodeModel models.NodeModel
		global.DB.Take(&nodeModel, honeyIP.NodeID)
		global.DB.Model(&nodeModel).Update("honey_ip_count", gorm.Expr("honey_ip_count + ?", netDeployInfo.AllCount))
		var netModel models.NetModel
		global.DB.Take(&netModel, honeyIP.NetID)
		global.DB.Model(&netModel).Update("honey_ip_count", gorm.Expr("honey_ip_count + ?", netDeployInfo.AllCount))
	}
}
