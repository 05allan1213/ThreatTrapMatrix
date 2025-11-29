package host_template_api

// File: image_server/api/host_template_api/list.go
// Description: 主机模板列表查询API接口

import (
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/service/common_service"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ListResponse 主机模板列表查询响应结构体
type ListResponse struct {
	models.HostTemplateModel                        // 嵌套主机模板基础信息
	PortList                 []HostTemplatePortInfo `json:"portList"` // 关联端口及服务信息列表
}

// HostTemplatePortInfo 主机模板端口详情结构体
type HostTemplatePortInfo struct {
	Port          int    `json:"port"`          // 端口号
	ServiceID     uint   `json:"serviceID"`     // 关联虚拟服务ID
	ServiceTitle  string `json:"serviceTitle"`  // 关联虚拟服务名称
	ServiceStatus int8   `json:"serviceStatus"` // 关联虚拟服务状态
}

// ListView 主机模板列表查询接口处理函数
func (HostTemplateApi) ListView(c *gin.Context) {
	// 获取并绑定分页查询参数
	cr := middleware.GetBind[models.PageInfo](c)

	// 调用公共查询服务分页查询主机模板列表
	_list, count, _ := common_service.QueryList(models.HostTemplateModel{},
		common_service.QueryListRequest{
			Likes:    []string{"title"}, // title字段支持模糊查询
			PageInfo: cr,                // 分页参数
			Sort:     "created_at desc", // 按创建时间降序排序
		})

	// 初始化响应列表
	var list = make([]ListResponse, 0)
	// 收集所有关联的虚拟服务ID（用于批量查询）
	var serviceList []models.ServiceModel
	var serviceIDList []uint
	for _, model := range _list {
		for _, port := range model.PortList {
			serviceIDList = append(serviceIDList, port.ServiceID)
		}
	}

	// 批量查询关联的虚拟服务信息
	global.DB.Find(&serviceList, "id in ?", serviceIDList)
	// 构建虚拟服务ID到服务模型的映射（便于快速匹配）
	var serviceMap = map[uint]models.ServiceModel{}
	for _, i2 := range serviceList {
		serviceMap[i2.ID] = i2
	}

	// 组装响应数据（关联虚拟服务信息）
	for _, model := range _list {
		portList := make([]HostTemplatePortInfo, 0)
		for _, port := range model.PortList {
			portList = append(portList, HostTemplatePortInfo{
				Port:          port.Port,
				ServiceID:     port.ServiceID,
				ServiceTitle:  serviceMap[port.ServiceID].Title,
				ServiceStatus: serviceMap[port.ServiceID].Status,
			})
		}
		list = append(list, ListResponse{
			HostTemplateModel: model,
			PortList:          portList,
		})
	}

	// 返回分页列表数据
	response.OkWithList(list, count, c)
}
