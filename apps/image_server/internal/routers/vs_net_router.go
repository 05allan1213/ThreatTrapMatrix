package routers

// File: image_server/routers/vs_net_router.go
// Description: 虚拟子网模块路由配置，定义虚拟子网相关接口的路由规则及中间件绑定

import (
	"image_server/internal/api"
	"image_server/internal/api/vs_net_api"
	"image_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// VsNetRouter 配置虚拟子网模块的路由规则
func VsNetRouter(r *gin.RouterGroup) {
	// 获取虚拟子网API接口实例
	app := api.App.VsNetApi

	// PUT /vs_net: 虚拟子网配置更新接口
	// 绑定JSON请求参数并处理配置更新逻辑
	r.PUT("vs_net", middleware.BindJsonMiddleware[vs_net_api.VsNetRequest], app.VsNetUpdateView)
	// GET /vs_net: 虚拟子网配置信息查询接口
	r.GET("vs_net", app.VsNetInfoView)
}
