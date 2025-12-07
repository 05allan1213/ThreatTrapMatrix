package api

// File: matrix_server/api/remove_deploy.go
// Description: 实现子网IP部署删除API接口

import (
	"errors"
	"matrix_server/internal/global"
	"matrix_server/internal/middleware"
	"matrix_server/internal/models"
	"matrix_server/internal/service/mq_service"
	"matrix_server/internal/service/redis_service/net_lock"
	"matrix_server/internal/service/redis_service/net_progress"
	"matrix_server/internal/utils/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RemoveDeployRequest 删除部署接口的请求参数结构体
type RemoveDeployRequest struct {
	IpList []string `json:"ipList" binding:"required,dive,ip"` // 待删除部署的IP列表
	NetID  uint     `json:"netID" binding:"required"`          // 子网ID
}

// RemoveDeployView 删除部署接口处理函数
func (Api) RemoveDeployView(c *gin.Context) {
	// 绑定并解析请求参数到RemoveDeployRequest结构体
	cr := middleware.GetBind[RemoveDeployRequest](c)
	log := middleware.GetLog(c)

	log.WithFields(map[string]interface{}{
		"net_id":   cr.NetID,
		"ip_count": len(cr.IpList),
	}).Info("batch remove deployment request received") // 收到批量删除部署请求
	// 校验IP列表是否为空
	if len(cr.IpList) == 0 {
		log.Warn("no IPs selected for removal") // 没有选择任何IP进行删除部署
		response.FailWithMsg("需要选择一个ip进行删除部署", c)
		return
	}

	// 查询子网信息并预加载关联的节点信息
	var model models.NetModel
	if err := global.DB.Preload("NodeModel").Take(&model, cr.NetID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("subnet not found") // 子网不存在
		response.FailWithMsg("子网不存在", c)
		return
	}
	// 1. 校验节点在线状态
	node := model.NodeModel
	if node.Status != 1 {
		log.WithFields(map[string]interface{}{
			"node_id":  node.ID,
			"node_uid": node.Uid,
			"status":   node.Status,
		}).Warn("node is offline") // 节点未运行
		response.FailWithMsg("节点离线", c)
		return
	}

	// 查询子网下指定IP且状态为已部署/部署中的蜜罐IP记录
	var honeyIpList []models.HoneyIpModel
	if err := global.DB.Find(
		&honeyIpList,
		"net_id = ? and ip in ? and status in ?",
		cr.NetID,
		cr.IpList,
		[]int8{2, 3}, // 假设2:已部署, 3:部署异常
	).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"ips":    cr.IpList,
			"error":  err,
		}).Error("failed to query honey IPs") // 查询诱捕IP列表失败
		response.FailWithMsg("查询诱捕IP信息失败", c)
		return
	}
	// 校验所有请求的IP均为已部署状态
	if len(honeyIpList) != len(cr.IpList) {
		log.WithFields(map[string]interface{}{
			"requested_count": len(cr.IpList),
			"valid_count":     len(honeyIpList),
		}).Warn("mismatch in valid honey IPs") // 有效诱捕IP数量不匹配
		response.FailWithMsg("存在未部署的ip", c)
		return
	}

	// 获取上下文日志实例及日志ID
	logID := log.Data["logID"].(string)
	// 组装MQ批量删除部署请求数据
	var batchRemoveData = mq_service.BatchRemoveDeployRequest{
		NetID: cr.NetID,
		LogID: logID,
		TanIp: model.IP,
	}

	// 2. 组装IP列表并写入Redis记录部署状态
	for _, ipModel := range honeyIpList {
		batchRemoveData.IPList = append(batchRemoveData.IPList, mq_service.RemoveDeployIp{
			Ip:       ipModel.IP,
			LinkName: ipModel.Network,
		})
		log.WithFields(map[string]interface{}{
			"ip":          ipModel.IP,
			"honey_ip_id": ipModel.ID,
		}).Debug("added IP to removal list") // 添加IP到删除列表
	}

	if err := net_lock.Lock(cr.NetID); err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("failed to acquire network lock") // 锁定子网失败
		response.FailWithMsg("当前子网正在部署中", c)
		return
	}

	// 3. 事务处理：更新IP状态并下发MQ删除部署消息
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 将诱捕IP状态更新为删除中（状态4）
		if err := tx.Model(&honeyIpList).Update("status", 4).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"error": err,
			}).Error("failed to update honey IP status") // 批量更新状态失败
			return errors.New("批量更新状态失败")
		}
		// 记录批量删除部署的IP数量日志
		log.WithFields(map[string]interface{}{
			"updated_count": len(honeyIpList),
		}).Info("honey IPs marked for removal") // 诱捕ip被标记为删除

		// Set removal progress tracking
		if err := net_progress.Set(cr.NetID, net_progress.NetDeployInfo{
			Type:     3, // 假设3:删除部署
			AllCount: int64(len(batchRemoveData.IPList)),
		}); err != nil {
			log.WithFields(map[string]interface{}{
				"net_id": cr.NetID,
				"error":  err,
			}).Error("failed to set removal progress") // 设置操作进度失败
			return errors.New("设置操作进度失败")
		}
		// 向MQ下发批量删除部署请求消息
		if err := mq_service.SendBatchRemoveDeployMsg(node.Uid, batchRemoveData); err != nil {
			log.WithFields(map[string]interface{}{
				"node_uid": node.Uid,
				"error":    err,
			}).Error("failed to send removal message") // 发送删除部署消息失败
			return errors.New("删除部署消息下发失败")
		}
		return nil
	})
	if err != nil {
		// 记录部署失败日志
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("removal transaction failed") // 删除部署事务执行失败
		net_lock.UnLock(cr.NetID)
		response.FailWithError(err, c)
		return
	}

	log.WithFields(map[string]interface{}{
		"net_id":       cr.NetID,
		"removing_ips": len(batchRemoveData.IPList),
	}).Info("batch removal initiated successfully") // 批量删除部署启动
	response.OkWithMsg("批量删除部署成功，正在删除中", c)
}
