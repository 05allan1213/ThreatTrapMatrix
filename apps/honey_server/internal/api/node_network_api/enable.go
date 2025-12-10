package node_network_api

// File: honey_server/api/node_network_api/enable.go
// Description: 节点网卡启用API接口

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/ip"
	"honey_server/internal/utils/response"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 互斥锁，用于控制并发访问
var mutex sync.Mutex

// EnableView 处理网卡启用的HTTP请求
func (NodeNetworkApi) EnableView(c *gin.Context) {
	log := middleware.GetLog(c)
	// 从请求中绑定并获取ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	log.WithFields(map[string]interface{}{
		"network_id": cr.Id,
	}).Info("network interface enable request received") // 收到网卡启用请求

	var model models.NodeNetworkModel
	// 查询指定ID的网卡信息，并预加载关联的节点模型
	if err := global.DB.Preload("NodeModel").Take(&model, cr.Id).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"network_id": cr.Id,
			"error":      err,
		}).Warn("network interface not found") // 网卡不存在
		response.FailWithMsg("网卡不存在", c)
		return
	}

	// 加锁确保并发安全，防止重复操作
	mutex.Lock()
	defer mutex.Unlock()

	// 检查网卡当前状态，已启用则直接返回错误
	if model.Status == 1 {
		log.WithFields(map[string]interface{}{
			"network_id": cr.Id,
			"status":     model.Status,
		}).Warn("network interface already enabled") // 网卡已启用
		response.FailWithMsg("网卡已启用，请勿重复启用", c)
		return
	}

	// 使用数据库事务执行启用逻辑，保证数据一致性
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		ipRange, err := ip.ParseCIDRGetUseIPRange(fmt.Sprintf("%s/%d", model.IP, model.Mask))
		if err != nil {
			log.WithFields(map[string]interface{}{
				"network_id": cr.Id,
				"ip":         model.IP,
				"mask":       model.Mask,
				"error":      err,
			}).Error("failed to parse IP range") // 解析IP范围失败
			return err
		}
		// 构建网络表记录，关联节点和网卡信息
		net := models.NetModel{
			NodeID:             model.NodeID,
			Title:              fmt.Sprintf("%s_%s_网络", model.NodeModel.Title, model.Network),
			Network:            model.Network,
			IP:                 model.IP,
			Mask:               model.Mask,
			Gateway:            model.Gateway,
			CanUseHoneyIPRange: ipRange,
		}
		// 插入网络表记录
		log.WithFields(map[string]interface{}{
			"network_id": cr.Id,
			"net_title":  net.Title,
			"ip_range":   ipRange,
		}).Info("creating network record") // 创建网络记录

		if err := tx.Create(&net).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"network_id": cr.Id,
				"error":      err,
			}).Error("failed to create network record") // 创建网络记录失败
			return err
		}

		// 更新网卡状态为已启用
		log.WithFields(map[string]interface{}{
			"network_id": cr.Id,
			"new_status": 1,
		}).Info("updating network interface status") // 更新网卡状态为已启用

		if err := tx.Model(&model).Updates(map[string]any{
			"status": 1,
			"net_id": net.ID,
		}).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"network_id": cr.Id,
				"error":      err,
			}).Error("failed to update network interface status") // 数据库更新网卡状态失败
			return err
		}
		return nil
	})

	if err != nil {
		log.WithFields(map[string]interface{}{
			"network_id": cr.Id,
			"error":      err,
		}).Error("transaction failed") // 事务失败
		response.FailWithMsg("网卡启用失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"network_id": cr.Id,
	}).Info("network interface enabled successfully") // 网卡启用成功
	response.OkWithMsg("网卡启用成功", c)
}
