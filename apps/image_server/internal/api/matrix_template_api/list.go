package matrix_template_api

// File: image_server/api/matrix_template_api/list.go
// Description: 矩阵模板列表查询API接口

import (
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/service/common_service"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ListResponse 矩阵模板列表查询响应结构体
type ListResponse struct {
	models.MatrixTemplateModel                    // 嵌套矩阵模板基础信息
	HostTemplateList           []HostTemplateInfo `json:"hostTemplateList"` // 关联主机模板详情列表
}

// HostTemplateInfo 矩阵模板关联的主机模板详情结构体
type HostTemplateInfo struct {
	HostTemplateID    uint   `json:"hostTemplateID"`    // 主机模板ID
	HostTemplateTitle string `json:"hostTemplateTitle"` // 主机模板标题
	Weight            int    `json:"weight"`            // 权重值
}

// ListView 矩阵模板列表查询接口处理函数
// 分页查询矩阵模板，并关联查询主机模板信息组装完整响应数据
func (MatrixTemplateApi) ListView(c *gin.Context) {
	// 获取并绑定分页查询参数
	cr := middleware.GetBind[models.PageInfo](c)

	// 调用公共查询服务分页查询矩阵模板列表
	_list, count, _ := common_service.QueryList(models.MatrixTemplateModel{},
		common_service.QueryListRequest{
			Likes:    []string{"title"}, // title字段支持模糊查询
			PageInfo: cr,                // 分页参数
			Sort:     "created_at desc", // 按创建时间降序排序
		})

	// 初始化响应列表
	var list = make([]ListResponse, 0)
	// 收集所有关联的主机模板ID（用于批量查询）
	var hostTemps []models.HostTemplateModel
	var hostTempIDList []uint
	for _, model := range _list {
		for _, port := range model.HostTemplateList {
			hostTempIDList = append(hostTempIDList, port.HostTemplateID)
		}
	}

	// 批量查询关联的主机模板信息
	global.DB.Find(&hostTemps, "id in ?", hostTempIDList)
	// 构建主机模板ID到模板模型的映射（便于快速匹配）
	var hostTempMap = map[uint]models.HostTemplateModel{}
	for _, i2 := range hostTemps {
		hostTempMap[i2.ID] = i2
	}

	// 组装响应数据（关联主机模板标题信息）
	for _, model := range _list {
		hostTemplateList := make([]HostTemplateInfo, 0)
		for _, port := range model.HostTemplateList {
			hostTemplateList = append(hostTemplateList, HostTemplateInfo{
				HostTemplateID:    port.HostTemplateID,
				HostTemplateTitle: hostTempMap[port.HostTemplateID].Title,
				Weight:            port.Weight,
			})
		}
		list = append(list, ListResponse{
			MatrixTemplateModel: model,
			HostTemplateList:    hostTemplateList,
		})
	}

	// 返回分页列表数据
	response.OkWithList(list, count, c)
}
