package host_template_api

// File: image_server/api/host_template_api/detail.go
// Description: 主机模板详情API接口

import (
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// DetailView 主机模板详情查询接口处理函数
func (HostTemplateApi) DetailView(c *gin.Context) {
	// 从Gin上下文绑定并解析ID请求参数
	cr := middleware.GetBind[models.IDRequest](c)

	// 定义主机模板模型变量，用于接收数据库查询结果
	var model models.HostTemplateModel
	// 根据模板ID从数据库查询单条主机模板记录
	err := global.DB.Take(&model, cr.ID).Error
	// 查询失败（无匹配记录）时返回错误提示
	if err != nil {
		response.FailWithMsg("主机模板不存在", c)
		return
	}

	// 查询成功，返回主机模板详情数据
	response.OkWithData(model, c)
}
