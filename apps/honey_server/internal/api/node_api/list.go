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

// ListRequest 节点列表分页查询请求参数结构体
type ListRequest struct {
	models.PageInfo
	NodeID uint `form:"nodeID"` // 节点ID
}

// ListView 节点列表分页查询接口处理函数
func (NodeApi) ListView(c *gin.Context) {
	// 从请求中绑定分页参数
	cr := middleware.GetBind[ListRequest](c)
	nodeModel := models.NodeModel{}
	nodeModel.ID = cr.NodeID
	list, count, _ := common_service.QueryList(nodeModel, common_service.QueryListRequest{
		Likes:    []string{"title", "ip"}, // 支持按节点名称、IP模糊搜索
		PageInfo: cr.PageInfo,             // 分页与搜索参数
		Sort:     "created_at desc",       // 排序规则
	})

	// 返回标准化分页响应
	response.OkWithList(list, count, c)
}
