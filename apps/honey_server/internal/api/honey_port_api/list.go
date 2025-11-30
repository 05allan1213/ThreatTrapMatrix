package honey_port_api

// File: honey_server/api/honey_port_api/list.go
// Description: 诱捕转发列表查询API接口

import (
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ListRequest 诱捕转发列表查询请求参数结构体
type ListRequest struct {
	models.PageInfo
	HoneyIPID uint `form:"honeyIpID" binding:"required"` // 关联的诱捕ipID（必填）
}

// ListResponse 诱捕转发列表查询响应结构体
type ListResponse struct {
	models.HoneyPortModel        // 诱捕转发基础信息模型
	ServiceTitle          string `json:"serviceTitle"` // 关联服务名称
}

// ListView 处理指定诱捕IP下的端口列表分页查询请求
func (HoneyPortApi) ListView(c *gin.Context) {
	// 获取并绑定请求参数（包含必填的诱捕ipID）
	cr := middleware.GetBind[ListRequest](c)

	// 调用通用查询服务获取端口列表及总数
	_list, count, _ := common_service.QueryList(models.HoneyPortModel{HoneyIpID: cr.HoneyIPID}, common_service.QueryListRequest{
		PageInfo: cr.PageInfo,              // 分页参数
		Sort:     "created_at desc",        // 按创建时间降序排序
		Preload:  []string{"ServiceModel"}, // 预加载关联的服务模型
	})

	// 转换数据格式，补充关联服务名称
	var list = make([]ListResponse, 0)
	for _, model := range _list {
		list = append(list, ListResponse{
			HoneyPortModel: model,
			ServiceTitle:   model.ServiceModel.Title, // 从关联服务模型获取名称
		})
	}

	// 返回分页列表数据
	response.OkWithList(list, count, c)
}
