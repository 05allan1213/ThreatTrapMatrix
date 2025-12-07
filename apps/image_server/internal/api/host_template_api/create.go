package host_template_api

// File: image_server/api/host_template_api/create.go
// Description: 主机模板创建API接口

import (
	"fmt"
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// CreateRequest 主机模板创建请求参数结构体
type CreateRequest struct {
	Title    string                      `json:"title" binding:"required"` // 主机模板名称（必需）
	PortList models.HostTemplatePortList `json:"portList" binding:"dive"`  // 端口列表（需校验端口唯一性及服务有效性）
}

// CreateView 主机模板创建接口处理函数
func (HostTemplateApi) CreateView(c *gin.Context) {
	// 获取日志句柄
	log := middleware.GetLog(c)
	// 获取并绑定主机模板创建请求参数
	cr := middleware.GetBind[CreateRequest](c)

	log.WithFields(map[string]interface{}{
		"request_data": cr,
	}).Info("host template creation request received") // 收到主机模板创建请求

	// 检查主机模板名称是否重复
	var existingModel models.HostTemplateModel
	if err := global.DB.Take(&existingModel, "title = ?", cr.Title).Error; err == nil {
		log.WithFields(map[string]interface{}{
			"title": cr.Title,
		}).Warn("duplicate host template title found") // 找到重复的主机模板名称
		response.FailWithMsg("主机模板名称不能重复", c)
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
			"port_count":        len(cr.PortList),
			"unique_port_count": len(portMap),
		}).Warn("duplicate ports detected in request") // 找到重复的端口
		response.FailWithMsg("端口存在重复", c)
		return
	}

	// 查询关联的虚拟服务记录并构建映射
	var serviceList []models.ServiceModel
	if err := global.DB.Find(&serviceList, "id in ?", serviceIDList).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"service_ids": serviceIDList,
			"error":       err,
		}).Error("failed to query services by IDs") // 查询虚拟服务失败
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
				"service_id":    port.ServiceID,
				"requested_ids": serviceIDList,
			}).Warn("invalid service ID provided") // 提供的虚拟服务ID不存在
			response.FailWithMsg(fmt.Sprintf("虚拟服务 %d 不存在", port.ServiceID), c)
			return
		}
	}

	// 组装主机模板数据并入库
	model := models.HostTemplateModel{
		Title:    cr.Title,
		PortList: cr.PortList,
	}
	if err := global.DB.Create(&model).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"model_data": model,
			"error":      err,
		}).Error("failed to create host template in database") // 主机模板入库失败
		response.FailWithMsg("主机模板创建失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"template_id": model.ID,
		"title":       model.Title,
	}).Info("host template created successfully") // 主机模板创建成功

	// 返回创建成功的模板ID
	response.OkWithData(model.ID, c)
}
