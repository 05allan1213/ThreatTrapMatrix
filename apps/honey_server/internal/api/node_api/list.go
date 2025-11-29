package node_api

// File: honey_server/api/node_api/list.go
// Description: 节点列表API接口

import (
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ListView 节点列表分页查询接口处理函数
func (NodeApi) ListView(c *gin.Context) {
	// 从请求中绑定分页参数（包含Page页码、PageSize页大小、Keyword搜索关键词）
	cr := middleware.GetBind[models.PageInfo](c)

	// 调用公共查询服务，获取节点列表数据与总数
	list, count, _ := common_service.QueryList(models.NodeModel{}, common_service.QueryListRequest{
		Likes:    []string{"title", "ip"}, // 支持按节点名称、IP模糊搜索
		PageInfo: cr,                      // 分页与搜索参数
		Sort:     "created_at desc",       // 排序规则
	})

	// 返回标准化分页响应
	response.OkWithList(list, count, c)
}
