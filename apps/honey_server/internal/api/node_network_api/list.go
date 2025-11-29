package node_network_api

// File: honey_server/api/node_network_api/list.go
// Description: 节点网卡列表查询API接口

import (
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ListRequest 网卡列表查询请求参数结构体
type ListRequest struct {
	NodeID          uint `form:"nodeID" binding:"required"` // 节点ID(必填)
	models.PageInfo                                         // 分页信息嵌套结构体（包含Page、PageSize字段）
}

// ListView 处理节点网卡列表查询请求
func (NodeNetworkApi) ListView(c *gin.Context) {
	// 绑定并验证请求参数
	cr := middleware.GetBind[ListRequest](c)

	// 调用通用查询服务获取网卡列表及总数
	list, count, _ := common_service.QueryList(models.NodeNetworkModel{NodeID: cr.NodeID}, common_service.QueryListRequest{
		Likes:    []string{"network", "ip"}, // 支持模糊搜索的字段
		PageInfo: cr.PageInfo,               // 分页参数
		Sort:     "created_at desc",         // 排序规则
	})

	// 返回分页列表数据响应
	response.OkWithList(list, count, c)
}
