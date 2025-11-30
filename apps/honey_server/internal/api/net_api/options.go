package net_api

// File: honey_server/api/net_api/options.go
// Description: 网络模块选项API接口

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// OptionsResponse 网络选项响应结构体
type OptionsResponse struct {
	Label string `json:"label"` // 显示文本(网络名称和子网信息)
	Value uint   `json:"value"` // 选项值(网络ID)
}

// OptionsView 处理网络选项查询请求，返回适配选择组件的网络列表
func (NetApi) OptionsView(c *gin.Context) {
	// 查询所有网络记录
	var netList []models.NetModel
	global.DB.Find(&netList)

	// 组装选项数据，格式化显示文本
	var list = make([]OptionsResponse, 0)
	for _, model := range netList {
		list = append(list, OptionsResponse{
			Value: model.ID,
			Label: fmt.Sprintf("%s(%s)", model.Title, model.Subnet()), // 组合标题和子网信息作为显示标签
		})
	}

	// 返回选项列表数据
	response.OkWithData(list, c)
}
