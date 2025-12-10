package vs_api

// File: image_server/api/vs_api/vs_list.go
// Description: 虚拟服务列表查询API接口

import (
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/service/common_service"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// VsListRequest 虚拟服务列表查询请求结构体
type VsListRequest struct {
	models.PageInfo        // 分页信息
	Port            int    `form:"port"` // 筛选条件：虚拟服务端口
	IP              string `form:"ip"` // 筛选条件：虚拟服务IP地址
	Title           string `form:"title"` // 筛选条件：虚拟服务标题（支持模糊搜索）
	VsID            uint   `form:"vsID"` // 筛选条件：虚拟服务ID（精准匹配）
}

// VsListView 虚拟服务列表查询接口处理函数
func (VsApi) VsListView(c *gin.Context) {
	// 绑定并解析前端提交的查询参数
	cr := middleware.GetBind[VsListRequest](c)

	// 构建虚拟服务筛选模型，映射查询参数到模型字段
	model := models.ServiceModel{
		Title: cr.Title, // 标题筛选
		IP:    cr.IP,    // IP地址筛选
		Port:  cr.Port,  // 端口筛选
	}
	model.ID = cr.VsID // 虚拟服务ID精准筛选

	// 调用通用查询服务获取列表数据
	list, count, _ := common_service.QueryList(model, common_service.QueryListRequest{
		Likes:    []string{"title"}, // 模糊搜索字段：title
		PageInfo: cr.PageInfo,       // 分页参数
		Sort:     "created_at desc", // 排序规则：按创建时间降序
	})

	// 返回分页列表数据（列表内容、总条数）
	response.OkWithList(list, count, c)
}
