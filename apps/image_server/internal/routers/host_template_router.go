package routers

// File: image_server/routers/host_template_routers.go
// Description: 主机模板模块路由配置，定义主机模板相关接口的路由规则及中间件绑定

import (
	"ThreatTrapMatrix/apps/image_server/internal/api"
	"ThreatTrapMatrix/apps/image_server/internal/api/host_template_api"
	"ThreatTrapMatrix/apps/image_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// HostTemplateRouter 配置主机模板模块的路由规则
func HostTemplateRouter(r *gin.RouterGroup) {
	// 获取主机模板API接口实例
	app := api.App.HostTemplateApi

	// POST /host_template: 主机模板创建接口
	// 绑定JSON请求参数并处理创建逻辑
	r.POST("host_template", middleware.BindJsonMiddleware[host_template_api.CreateRequest], app.CreateView)
}
