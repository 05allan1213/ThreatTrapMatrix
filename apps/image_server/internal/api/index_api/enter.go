package index_api

// File: image_server/api/index_api/enter.go
// Description: 首页模块API接口

import (
	"image_server/internal/global"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// IndexApi 首页模块API接口结构体，封装首页数据统计相关接口方法
type IndexApi struct {
}

// IndexCountResponse 首页统计数据响应结构体
type IndexCountResponse struct {
	ImageCount int64 `json:"imageCount"` // 图片总数
	VsCount    int64 `json:"vsCount"`    // 服务总数
}

// IndexCountView 首页统计数据查询接口处理函数
func (IndexApi) IndexCountView(c *gin.Context) {
	// 初始化统计响应结构体
	var data IndexCountResponse

	// 查询图片表总记录数
	global.DB.Model(models.ImageModel{}).Count(&data.ImageCount)
	// 查询服务表总记录数
	global.DB.Model(models.ServiceModel{}).Count(&data.VsCount)

	// 返回成功响应，携带首页统计数据
	response.OkWithData(data, c)
}
