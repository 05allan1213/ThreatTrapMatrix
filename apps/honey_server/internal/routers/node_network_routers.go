package routers

// File: honey_server/routers/node_network_routers.go
// Description: 节点网卡模块路由配置，注册网卡相关API接口路由

import (
	"honey_server/internal/api"
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
}
