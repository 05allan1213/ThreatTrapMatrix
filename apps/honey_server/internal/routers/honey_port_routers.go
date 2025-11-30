package routers

// File: honey_server/routers/honey_port_routers.go
// Description: 诱捕端口模块路由配置，定义诱捕端口相关API接口的路由规则与中间件绑定

import (
	"honey_server/internal/api"
	"honey_server/internal/api/honey_port_api"
	"honey_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// HoneyPortRouters 配置诱捕端口模块的路由规则
func HoneyPortRouters(r *gin.RouterGroup) {
	// 获取诱捕端口API接口实例
	var app = api.App.HoneyPortApi
	// PUT /honey_port: 诱捕端口更新接口
	// 绑定JSON数据到UpdateRequest结构体，并调用UpdateView方法处理请求
	r.PUT("honey_port", middleware.BindJsonMiddleware[honey_port_api.UpdateRequest], app.UpdateView)
	// GET /honey_port: 获取诱捕端口列表接口
	// 绑定Query参数到ListRequest结构体，并调用ListView方法处理请求
	r.GET("honey_port", middleware.BindQueryMiddleware[honey_port_api.ListRequest], app.ListView)
}
