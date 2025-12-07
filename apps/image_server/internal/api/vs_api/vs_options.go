package vs_api

// File: image_server/api/vs_api/vs_options.go
// Description: 虚拟服务选项列表接口实现，提供前端下拉选择所需的虚拟服务选项数据

import (
	"fmt"
	"image_server/internal/global"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// VsOptionsListResponse 虚拟服务选项列表响应结构体
type VsOptionsListResponse struct {
	Label   string `json:"label"`   // 选项展示文本（服务名称+端口）
	Value   uint   `json:"value"`   // 选项值（虚拟服务ID）
	Disable bool   `json:"disable"` // 是否禁用选项（服务状态非运行中时禁用）
}

// VsOptionsListView 获取虚拟服务选项列表接口
func (VsApi) VsOptionsListView(c *gin.Context) {
	// 查询所有虚拟服务记录
	var list []models.ServiceModel
	global.DB.Find(&list)

	// 组装虚拟服务选项数据
	var options []VsOptionsListResponse
	for _, model := range list {
		item := VsOptionsListResponse{
			Label: fmt.Sprintf("%s/%d", model.Title, model.Port), // 拼接展示文本（名称/端口）
			Value: model.ID,
		}
		// 服务状态非运行中（1）时设置选项禁用
		if model.Status != 1 {
			item.Disable = true
		}
		options = append(options, item)
	}

	// 返回虚拟服务选项列表数据
	response.OkWithData(options, c)
}
