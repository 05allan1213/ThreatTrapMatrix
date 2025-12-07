package node_network_api

// File: honey_server/api/node_network_api/remove.go
// Description: 节点网卡删除API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// RemoveView 处理节点网卡删除请求
func (NodeNetworkApi) RemoveView(c *gin.Context) {
	log := middleware.GetLog(c)
	// 绑定并获取请求中的网卡ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	log.WithFields(map[string]interface{}{
		"network_id": cr.Id,
	}).Info("network interface deletion request received") // 收到节点网卡删除请求

	var model models.NodeNetworkModel
	// 查询指定ID的网卡记录是否存在
	if err := global.DB.Take(&model, cr.Id).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"network_id": cr.Id,
			"error":      err,
		}).Warn("network interface not found") // 网卡不存在
		response.FailWithMsg("网卡不存在", c)
		return
	}

	// 执行网卡记录删除操作
	if err := global.DB.Delete(&model).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"network_id": cr.Id,
			"error":      err,
		}).Error("failed to delete network interface") // 网卡删除失败
		response.FailWithMsg("网卡删除失败"+err.Error(), c)
		return
	}

	log.WithFields(map[string]interface{}{
		"network_id": cr.Id,
	}).Info("network interface deleted successfully") // 网卡删除成功
	response.OkWithMsg("网卡删除成功", c)
}
