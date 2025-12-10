package net_api

// File: honey_server/api/net_api/list.go
// Description: 网络模块列表查询API接口

import (
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ListRequest 网络列表查询请求参数结构体
type ListRequest struct {
	NodeID          uint `form:"nodeID"` // 节点ID
	NetID           uint `form:"netID"`  // 网络ID
	models.PageInfo                      // 分页信息嵌套结构体
}

// ListResponse 网络列表查询响应结构体，扩展节点关联信息
type ListResponse struct {
	models.NetModel        // 网络基础信息模型
	NodeTitle       string `json:"nodeTitle"` // 关联节点名称
	NodeStatus      int8   `json:"nodeStatus"` // 关联节点状态
}

// ListView 处理网络列表查询请求，支持按节点筛选及分页
func (NetApi) ListView(c *gin.Context) {
	// 绑定并解析请求参数
	cr := middleware.GetBind[ListRequest](c)

	// 调用通用查询服务获取网络列表，预加载关联的节点模型
	model := models.NetModel{NodeID: cr.NodeID}
	model.ID = cr.NetID

	_list, count, _ := common_service.QueryList(model, common_service.QueryListRequest{
		Likes:    []string{"title", "ip"}, // 支持标题和IP的模糊搜索
		PageInfo: cr.PageInfo,             // 分页参数
		Sort:     "created_at desc",       // 按创建时间降序排序
		Preload:  []string{"NodeModel"},   // 预加载节点关联数据
	})

	// 组装响应数据，补充节点关联信息
	var list = make([]ListResponse, 0)
	for _, model := range _list {
		// 获取当前网络扫描进度
		_progress, ok := netProgressMap.Load(model.ID)
		if ok {
			progress := _progress.(float64)
			model.ScanProgress = progress
		}
		list = append(list, ListResponse{
			NetModel:   model,
			NodeTitle:  model.NodeModel.Title,  // 从关联节点模型获取名称
			NodeStatus: model.NodeModel.Status, // 从关联节点模型获取状态
		})
	}

	// 返回分页列表响应
	response.OkWithList(list, count, c)
}
