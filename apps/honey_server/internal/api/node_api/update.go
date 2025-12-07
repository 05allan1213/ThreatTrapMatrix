package node_api

// File: honey_server/api/node_api/update.go
// Description: 节点信息更新API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// UpdateRequest 节点更新请求参数结构体
type UpdateRequest struct {
	ID    uint   `json:"id" binding:"required"`    // 节点ID（必需）
	Title string `json:"title" binding:"required"` // 节点新名称（必需）
}

// UpdateView 节点信息更新接口处理函数
func (NodeApi) UpdateView(c *gin.Context) {
	log := middleware.GetLog(c)
	// 绑定请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	log.WithFields(map[string]interface{}{
		"node_id":   cr.ID,
		"new_title": cr.Title,
	}).Info("node update request received") // 收到节点更新请求

	// 查询待更新节点是否存在
	var model models.NodeModel
	if err := global.DB.Take(&model, cr.ID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"node_id": cr.ID,
			"error":   err,
		}).Warn("node not found for update") // 节点不存在
		response.FailWithMsg("节点不存在", c)
		return
	}

	// 更新节点名称（仅更新title字段）
	if err := global.DB.Model(&model).Update("title", cr.Title).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"node_id":   cr.ID,
			"new_title": cr.Title,
			"error":     err,
		}).Error("failed to update node title") // 节点修改失败
		response.FailWithMsg("节点修改失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"node_id":       cr.ID,
		"updated_title": cr.Title,
	}).Info("node title updated successfully") // 节点修改成功
	// 更新成功，返回提示信息
	response.OkWithMsg("节点修改成功", c)
}
