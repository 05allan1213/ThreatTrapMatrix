package host_template_api

// File: image_server/api/host_template_api/update.go
// Description: 主机模板更新接口实现，包含模板存在性校验、名称唯一性校验、端口及服务有效性校验

import (
	"fmt"
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// UpdateRequest 主机模板更新请求参数结构体
type UpdateRequest struct {
	ID       uint                        `json:"id" binding:"required"`    // 主机模板ID（必填）
	Title    string                      `json:"title" binding:"required"` // 新模板名称（需保证唯一性）
	PortList models.HostTemplatePortList `json:"portList" binding:"dive"`  // 更新后的端口列表（需校验端口唯一性及服务有效性）
}

// UpdateView 主机模板更新接口处理函数
func (HostTemplateApi) UpdateView(c *gin.Context) {
	log := middleware.GetLog(c)
	// 获取并绑定主机模板更新请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	log.WithFields(map[string]interface{}{
		"template_id":  cr.ID,
		"request_data": cr,
	}).Info("host template update request received") // 收到主机模板更新请求

	// 校验待更新的主机模板是否存在
	var model models.HostTemplateModel
	if err := global.DB.Take(&model, cr.ID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"template_id": cr.ID,
			"error":       err,
		}).Warn("host template not found") // 主机模板不存在
		response.FailWithMsg("主机模板不存在", c)
		return
	}

	// 校验新模板名称的唯一性（排除自身ID）
	var duplicateModel models.HostTemplateModel
	if err := global.DB.Take(&duplicateModel, "title = ? and id <> ?", cr.Title, cr.ID).Error; err == nil {
		log.WithFields(map[string]interface{}{
			"template_id":    cr.ID,
			"title":          cr.Title,
			"conflicting_id": duplicateModel.ID,
		}).Warn("duplicate host template title found") // 找到重复的主机模板名称
		response.FailWithMsg("修改的主机模板名称不能重复", c)
		return
	}

	// 校验端口唯一性及收集关联服务ID
	var serviceIDList []uint
	portMap := make(map[int]bool)
	for _, port := range cr.PortList {
		serviceIDList = append(serviceIDList, port.ServiceID)
		portMap[port.Port] = true // 用Map去重校验端口唯一性
	}

	// 检查是否存在重复端口
	if len(portMap) != len(cr.PortList) {
		log.WithFields(map[string]interface{}{
			"template_id":  cr.ID,
			"port_count":   len(cr.PortList),
			"unique_ports": len(portMap),
		}).Warn("duplicate ports detected in update request") // 存在重复端口
		response.FailWithMsg("端口存在重复", c)
		return
	}

	// 查询关联的虚拟服务记录并构建映射
	var serviceList []models.ServiceModel
	if err := global.DB.Find(&serviceList, "id in ?", serviceIDList).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"template_id": cr.ID,
			"service_ids": serviceIDList,
			"error":       err,
		}).Error("failed to query referenced services") // 查询关联的虚拟服务记录失败
		response.FailWithMsg("查询虚拟服务失败", c)
		return
	}

	// 构建虚拟服务映射
	serviceMap := make(map[uint]models.ServiceModel)
	for _, service := range serviceList {
		serviceMap[service.ID] = service
	}

	// 校验所有关联服务是否存在
	for _, port := range cr.PortList {
		if _, exists := serviceMap[port.ServiceID]; !exists {
			log.WithFields(map[string]interface{}{
				"template_id": cr.ID,
				"service_id":  port.ServiceID,
			}).Warn("referenced service does not exist") // 关联的虚拟服务不存在
			response.FailWithMsg(fmt.Sprintf("虚拟服务 %d 不存在", port.ServiceID), c)
			return
		}
	}

	// 组装更新数据并执行更新操作
	updateData := models.HostTemplateModel{
		Title:    cr.Title,
		PortList: cr.PortList,
	}
	if err := global.DB.Model(&model).Updates(updateData).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"template_id": cr.ID,
			"update_data": updateData,
			"error":       err,
		}).Error("failed to update host template") // 主机模板更新失败

		response.FailWithMsg("主机模板更新失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"template_id": cr.ID,
		"new_title":   cr.Title,
		"port_count":  len(cr.PortList),
	}).Info("host template updated successfully") // 主机模板更新成功

	response.OkWithMsg("主机模板更新成功", c)
}
