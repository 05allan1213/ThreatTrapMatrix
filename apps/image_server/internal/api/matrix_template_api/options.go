package matrix_template_api

// File: image_server/api/matrix_template_api/options.go
// Description: 矩阵模板选项列表API接口

import (
	"image_server/internal/global"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// OptionsListResponse 矩阵模板选项列表响应结构体
type OptionsListResponse struct {
	Label string `json:"label"` // 选项展示文本（矩阵模板名称）
	Value uint   `json:"value"` // 选项值（矩阵模板ID）
}

// OptionsView 获取矩阵模板选项列表接口
func (MatrixTemplateApi) OptionsView(c *gin.Context) {
	// 初始化选项列表
	var list = make([]OptionsListResponse, 0)

	// 查询矩阵模板的ID和名称，并映射到选项结构体
	global.DB.Model(models.MatrixTemplateModel{}).Select("id as value", "title as label").Scan(&list)

	// 返回选项列表数据
	response.OkWithData(list, c)
}
