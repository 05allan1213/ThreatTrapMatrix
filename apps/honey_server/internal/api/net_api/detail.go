package net_api

// File: honey_server/api/net_api/detail.go
// Description: 网络模块详情API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// DetailView 处理网络详情查询请求，返回指定ID的网络完整信息
func (NetApi) DetailView(c *gin.Context) {
	// 绑定并获取请求中的网络ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	var model models.NetModel
	// 查询指定ID的网络记录
	err := global.DB.Take(&model, cr.Id).Error
	if err != nil {
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 返回网络详情数据
	response.OkWithData(model, c)
}
