package host_template_api

// File: image_server/api/host_template_api/update.go
// Description: 主机模板更新接口实现，包含模板存在性校验、名称唯一性校验、端口及服务有效性校验

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/middleware"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"
	"fmt"

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
	// 获取并绑定主机模板更新请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	// 校验待更新的主机模板是否存在
	var model models.HostTemplateModel
	err := global.DB.Take(&model, cr.ID).Error
	if err != nil {
		response.FailWithMsg("主机模板不存在", c)
		return
	}

	// 校验新模板名称的唯一性（排除自身ID）
	var newModel models.HostTemplateModel
	err = global.DB.Take(&newModel, "title = ? and id <> ?", cr.Title, cr.ID).Error
	if err == nil {
		response.FailWithMsg("修改的主机模板名称不能重复", c)
		return
	}

	// 校验端口唯一性及收集关联服务ID
	var serviceIDList []uint
	var portMap = map[int]bool{}
	for _, port := range cr.PortList {
		serviceIDList = append(serviceIDList, port.ServiceID)
		portMap[port.Port] = true // 用Map去重校验端口唯一性
	}

	// 检查是否存在重复端口
	if len(portMap) != len(cr.PortList) {
		response.FailWithMsg("端口存在重复", c)
		return
	}

	// 查询关联的虚拟服务记录并构建映射
	var serviceList []models.ServiceModel
	global.DB.Find(&serviceList, "id in ?", serviceIDList)
	var serviceMap = map[uint]models.ServiceModel{}
	for _, serviceModel := range serviceList {
		serviceMap[serviceModel.ID] = serviceModel
	}

	// 校验所有关联服务是否存在
	for _, port := range cr.PortList {
		_, ok := serviceMap[port.ServiceID]
		if !ok {
			msg := fmt.Sprintf("虚拟服务 %d 不存在", port.ServiceID)
			response.FailWithMsg(msg, c)
			return
		}
	}

	// 组装更新数据并执行更新操作
	newModel = models.HostTemplateModel{
		Title:    cr.Title,
		PortList: cr.PortList,
	}
	err = global.DB.Model(&model).Updates(newModel).Error
	if err != nil {
		response.FailWithMsg("主机模板更新失败", c)
		return
	}

	response.OkWithMsg("主机模板更新成功", c)
}
