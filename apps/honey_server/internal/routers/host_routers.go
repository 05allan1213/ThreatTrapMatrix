package routers

// File: honey_server/routers/host_routers.go
// Description: 主机模块路由配置，定义主机相关API接口的路由规则与中间件绑定

import (
	"honey_server/internal/api"
	"honey_server/internal/api/host_api"
	"honey_server/internal/middleware"
	"honey_server/internal/models"

	"github.com/gin-gonic/gin"
)

// HostRouters 配置主机模块的路由规则
func HostRouters(r *gin.RouterGroup) {
	// 获取主机API接口实例
	var app = api.App.HostApi
	// GET /host: 主机列表查询接口
	// 使用Query参数绑定中间件解析主机列表查询请求参数
	r.GET("host", middleware.BindQueryMiddleware[host_api.ListRequest], app.ListView)
	// DELETE /host: 主机删除接口
	// 使用JSON参数绑定中间件解析主机删除请求参数
	r.DELETE("host", middleware.BindJsonMiddleware[models.IDListRequest], app.RemoveView)
}
