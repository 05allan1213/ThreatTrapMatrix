package routers

// File: image_server/routers/matrix_template_routers.go
// Description: 矩阵模板模块路由配置，定义矩阵模板相关接口的路由规则及中间件绑定

import (
	"image_server/internal/api"
	"image_server/internal/api/matrix_template_api"
	"image_server/internal/middleware"
	"image_server/internal/models"

	"github.com/gin-gonic/gin"
)

// MatrixTemplateRouter 配置矩阵模板模块的路由规则
func MatrixTemplateRouter(r *gin.RouterGroup) {
	// 获取矩阵模板API接口实例
	app := api.App.MatrixTemplateApi

	// POST /matrix_template: 矩阵模板创建接口
	// 绑定JSON请求参数并处理创建逻辑
	r.POST("matrix_template", middleware.BindJsonMiddleware[matrix_template_api.CreateRequest], app.CreateView)
	// PUT /matrix_template: 矩阵模板更新接口
	// 绑定JSON请求参数并处理更新逻辑
	r.PUT("matrix_template", middleware.BindJsonMiddleware[matrix_template_api.UpdateRequest], app.UpdateView)
	// GET /matrix_template: 矩阵模板列表查询接口
	// 绑定查询参数并处理列表查询
	r.GET("matrix_template", middleware.BindQueryMiddleware[models.PageInfo], app.ListView)
	// GET /matrix_template/options: 矩阵模板选项列表接口
	r.GET("matrix_template/options", app.OptionsView)
	// DELETE /matrix_template: 矩阵模板批量删除接口
	// 绑定JSON请求参数并处理删除逻辑
	r.DELETE("matrix_template", middleware.BindJsonMiddleware[models.IDListRequest], app.Remove)
}
