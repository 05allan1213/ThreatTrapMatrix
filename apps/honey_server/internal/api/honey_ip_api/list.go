package honey_ip_api

// File: honey_server/api/honey_ip_api/list.go
// Description: 诱捕IP列表查询API接口

import (
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ListRequest 诱捕IP列表查询请求参数结构体
type ListRequest struct {
	models.PageInfo      // 分页参数
	NodeID          uint `form:"nodeID"` // 节点id
	NetID           uint `form:"netID"` // 网络id
}

// ListResponse 诱捕IP列表查询响应结构体
type ListResponse struct {
	models.HoneyIpModel        // 诱捕IP基础信息模型
	NetTitle            string `json:"netTitle"` // 关联网络名称
	NodeTitle           string `json:"nodeTitle"` // 关联节点名称
}

// ListView 处理诱捕IP列表分页查询请求
func (HoneyIPApi) ListView(c *gin.Context) {
	// 获取并绑定请求参数
	cr := middleware.GetBind[ListRequest](c)

	// 调用通用查询服务获取诱捕IP列表及总数
	_list, count, _ := common_service.QueryList(models.HoneyIpModel{NodeID: cr.NodeID, NetID: cr.NetID}, common_service.QueryListRequest{
		Likes:    []string{"ip", "mac"},             // 支持IP和MAC地址模糊查询
		PageInfo: cr.PageInfo,                       // 分页参数
		Sort:     "created_at desc",                 // 按创建时间降序排序
		Preload:  []string{"NodeModel", "NetModel"}, // 预加载关联的节点和网络模型
	})

	// 转换数据格式，补充关联对象名称
	var list = make([]ListResponse, 0)
	for _, model := range _list {
		list = append(list, ListResponse{
			HoneyIpModel: model,
			NodeTitle:    model.NodeModel.Title, // 从关联节点模型获取名称
			NetTitle:     model.NetModel.Title,  // 从关联网络模型获取名称
		})
	}

	// 返回分页列表数据
	response.OkWithList(list, count, c)
}
