package node_api

// File: honey_server/api/node_api/options.go
// Description: 节点选项查询API接口

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// OptionsResponse 节点选项响应结构体
type OptionsResponse struct {
	Label string `json:"label"` // 显示文本（节点名称+IP）
	Value uint   `json:"value"` // 选中值（节点ID）
}

// OptionsView 节点选项接口处理函数
func (NodeApi) OptionsView(c *gin.Context) {
	// 查询数据库中所有节点记录
	var nodeList []models.NodeModel
	global.DB.Find(&nodeList)

	// 初始化返回列表（容量为0，动态扩展）
	var list = make([]OptionsResponse, 0)

	// 遍历节点列表，格式化为前端所需的下拉选项结构
	for _, model := range nodeList {
		list = append(list, OptionsResponse{
			Value: model.ID,                                     // 选中值(节点ID)
			Label: fmt.Sprintf("%s(%s)", model.Title, model.IP), // 显示文本(节点名称+IP)
		})
	}

	// 返回格式化后的下拉选项数据
	response.OkWithData(list, c)
}
