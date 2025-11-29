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
	// 获取并绑定主机模板创建请求参数
	cr := middleware.GetBind[CreateRequest](c)

	// 校验主机模板名称唯一性
	var model models.HostTemplateModel
	err := global.DB.Take(&model, "title = ? ", cr.Title).Error
	if err == nil {
		response.FailWithMsg("主机模板名称不能重复", c)
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

	// 组装主机模板数据并入库
	model = models.HostTemplateModel{
		Title:    cr.Title,
		PortList: cr.PortList,
	}
	err = global.DB.Create(&model).Error
	if err != nil {
		response.FailWithMsg("主机模板创建失败", c)
		return
	}

	// 返回创建成功的模板ID
	response.OkWithData(model.ID, c)
}
