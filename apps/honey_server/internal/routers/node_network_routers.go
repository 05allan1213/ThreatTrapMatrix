package routers

// File: honey_server/routers/node_network_routers.go
// Description: 节点网卡模块路由配置，注册网卡相关API接口路由

import (
	"honey_server/internal/api"
	"honey_server/internal/api/node_network_api"
	"honey_server/internal/middleware"
	"honey_server/internal/models"

	"github.com/gin-gonic/gin"
)

// NodeNetworkRouters 注册节点网卡相关API路由
func NodeNetworkRouters(r *gin.RouterGroup) {
	// 获取节点网卡API实例
	var app = api.App.NodeNetworkApi
	// GET /node_network/flush - 获取节点网卡信息刷新
	// 使用Query参数绑定中间件解析通用ID请求参数
	r.GET("node_network/flush", middleware.BindQueryMiddleware[models.IDRequest], app.FlushView)
	// GET /node_network - 获取节点网卡列表
	// 使用Query参数绑定中间件解析通用列表请求参数
	r.GET("node_network", middleware.BindQueryMiddleware[node_network_api.ListRequest], app.ListView)
	// PUT /node_network - 更新节点网卡信息
	// 使用JSON参数绑定中间件解析通用更新请求参数
	r.PUT("node_network", middleware.BindJsonMiddleware[node_network_api.UpdateRequest], app.UpdateView)
	// PUT /node_network/enable - 启用节点网卡
	// 使用JSON参数绑定中间件解析通用ID请求参数
	r.PUT("node_network/enable", middleware.BindJsonMiddleware[models.IDRequest], app.EnableView)
	// DELETE /node_network/:id - 删除节点网卡
	// 使用URI参数绑定中间件解析通用ID请求参数
	r.DELETE("node_network/:id", middleware.BindUriMiddleware[models.IDRequest], app.RemoveView)
}
