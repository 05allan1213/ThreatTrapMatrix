package routers

// File: image_server/routers/vs_router.go
// Description: 虚拟服务路由注册

import (
	"image_server/internal/api"
	"image_server/internal/api/vs_api"
	"image_server/internal/middleware"
	"image_server/internal/models"

	"github.com/gin-gonic/gin"
)

// VsRouter 配置虚拟服务模块的路由规则
func VsRouter(r *gin.RouterGroup) {
	// 获取虚拟服务API接口实例
	app := api.App.VsApi

	// POST /vs: 虚拟服务创建接口
	// 绑定JSON请求参数并处理创建逻辑
	r.POST("vs", middleware.BindJsonMiddleware[vs_api.VsCreateRequest], app.VsCreateView)
	// GET /vs: 虚拟服务列表查询接口
	// 绑定Query参数并处理列表逻辑
	r.GET("vs", middleware.BindQueryMiddleware[vs_api.VsListRequest], app.VsListView)
	// GET /vs/options: 虚拟服务选项列表接口
	r.GET("vs/options", app.VsOptionsListView)
	// DELETE /vs: 虚拟服务批量删除接口
	// 绑定JSON请求参数并处理删除逻辑
	r.DELETE("vs", middleware.BindJsonMiddleware[models.IDListRequest], app.VsRemoveView)
}
