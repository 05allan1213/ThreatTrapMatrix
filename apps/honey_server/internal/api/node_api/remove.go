package node_api

// File: honey_server/api/node_api/remove.go
// Description: 节点删除API接口实现

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// RemoveView 节点删除接口处理函数
func (NodeApi) RemoveView(c *gin.Context) {
	log := middleware.GetLog(c)
	// 从请求中绑定并获取节点ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	log.WithFields(map[string]interface{}{
		"node_id": cr.Id,
	}).Info("node deletion request received") // 收到节点删除请求

	// 查询待删除节点是否存在
	var model models.NodeModel
	if err := global.DB.Take(&model, cr.Id).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"node_id": cr.Id,
			"error":   err,
		}).Warn("node not found") // 节点不存在
		response.FailWithMsg("节点不存在", c)
		return
	}

	// 执行节点删除操作(物理删除)
	if err := global.DB.Delete(&model).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"node_id": cr.Id,
			"error":   err,
		}).Error("database deletion failed") // 数据库删除失败
		response.FailWithMsg("节点删除失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"node_id": cr.Id,
	}).Info("node deleted successfully") // 节点删除成功
	// 删除成功，返回提示信息
	response.OkWithMsg("节点删除成功", c)
}
