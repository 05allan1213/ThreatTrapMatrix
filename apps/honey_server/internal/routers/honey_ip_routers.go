package routers

// File: honey_server/routers/honey_ip_routers.go
// Description: 诱捕IP模块路由配置，定义诱捕IP相关API接口的路由规则与中间件绑定

import (
	"honey_server/internal/api"
	"honey_server/internal/api/honey_ip_api"
	"honey_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// HoneyIPRouters 配置诱捕IP模块的路由规则
func HoneyIPRouters(r *gin.RouterGroup) {
	// 获取诱捕IP API接口实例
	var app = api.App.HoneyIPApi
	// POST /honey_ip: 诱捕IP创建接口
	// 使用JSON参数绑定中间件解析诱捕IP创建请求参数
	r.POST("honey_ip", middleware.BindJsonMiddleware[honey_ip_api.CreateRequest], app.CreateView)
	// GET /honey_ip: 诱捕IP列表接口
	// 使用Query参数绑定中间件解析诱捕IP列表请求参数
	r.GET("honey_ip", middleware.BindQueryMiddleware[honey_ip_api.ListRequest], app.ListView)
	// DELETE /honey_ip: 诱捕IP删除接口
	// 使用JSON参数绑定中间件解析诱捕IP删除请求参数
	r.DELETE("honey_ip", middleware.BindJsonMiddleware[honey_ip_api.RemoveRequest], app.RemoveView)
}
