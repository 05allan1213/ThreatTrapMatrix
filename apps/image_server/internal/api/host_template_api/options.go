package host_template_api

// File: image_server/api/host_template_api/options.go
// Description: 主机模板选项列表API接口

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// OptionsListResponse 主机模板选项列表响应结构体
type OptionsListResponse struct {
	Label string `json:"label"` // 选项展示文本（主机模板名称）
	Value uint   `json:"value"` // 选项值（主机模板ID）
}

// OptionsView 获取主机模板选项列表接口
func (HostTemplateApi) OptionsView(c *gin.Context) {
	// 初始化选项列表
	var list = make([]OptionsListResponse, 0)

	// 查询主机模板的ID和名称，并映射到选项结构体
	global.DB.Model(models.HostTemplateModel{}).Select("id as value", "title as label").Scan(&list)

	// 返回选项列表数据
	response.OkWithData(list, c)
}
