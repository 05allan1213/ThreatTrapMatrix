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

// VsListRequest 虚拟服务列表查询请求参数结构体
type VsListRequest struct {
	models.PageInfo        // 嵌套分页参数（页码、页大小）
	Port            int    `form:"port"` // 筛选条件：服务端口
	IP              string `form:"ip"` // 筛选条件：容器IP地址
	Title           string `form:"title"` // 筛选条件：服务名称（支持模糊匹配）
}

// VsListView 虚拟服务列表查询接口处理函数
func (VsApi) VsListView(c *gin.Context) {
	// 获取并绑定列表查询请求参数（含分页和筛选条件）
	cr := middleware.GetBind[VsListRequest](c)

	// 调用公共查询服务，构建查询条件并执行分页查询
	list, count, _ := common_service.QueryList(models.ServiceModel{
		Title: cr.Title, // 名称精确/模糊匹配
		IP:    cr.IP,    // IP精确匹配
		Port:  cr.Port,  // 端口精确匹配
	},
		common_service.QueryListRequest{
			Likes:    []string{"title"}, // 指定title字段支持模糊查询
			PageInfo: cr.PageInfo,       // 分页参数
			Sort:     "created_at desc", // 排序规则：按创建时间降序
		})

	// 返回分页列表数据
	response.OkWithList(list, count, c)
}
