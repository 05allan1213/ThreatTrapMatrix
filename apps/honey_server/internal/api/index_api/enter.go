package index_api

// File: honey_server//api/index_api/enter.go
// Description: 首页统计API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// IndexApi 首页模块API接口结构体，封装首页数据统计相关接口方法
type IndexApi struct {
}

// IndexCountResponse 首页统计数据响应结构体
type IndexCountResponse struct {
	NodeCount    int64 `json:"nodeCount"`    // 节点总数
	NetCount     int64 `json:"netCount"`     // 网络总数
	HoneyIpCount int64 `json:"honeyIpCount"` // 蜜罐IP总数
}

// IndexCountView 首页统计数据查询接口处理函数
func (IndexApi) IndexCountView(c *gin.Context) {
	// 初始化统计响应结构体
	var data IndexCountResponse

	// 查询节点表总记录数
	global.DB.Model(models.NodeModel{}).Count(&data.NodeCount)
	// 查询网络表总记录数
	global.DB.Model(models.NetModel{}).Count(&data.NetCount)
	// 查询蜜罐IP表总记录数
	global.DB.Model(models.HoneyIpModel{}).Count(&data.HoneyIpCount)

	// 返回成功响应，携带首页统计数据
	response.OkWithData(data, c)
}
