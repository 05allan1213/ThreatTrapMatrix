package routers

// File: honey_server/routers/node_routers.go
// Description: 节点模块路由注册器，负责将节点相关的API接口绑定到指定路由，并配置参数解析中间件

import (
	"honey_server/internal/api"
	"honey_server/internal/api/node_api"
	"honey_server/internal/middleware"
	"honey_server/internal/models"

	"github.com/gin-gonic/gin"
)

// NodeRouters 注册节点模块的路由
func NodeRouters(r *gin.RouterGroup) {
	// 获取节点API处理器实例
	var app = api.App.NodeApi

	// GET /node - 获取节点列表
	// 绑定Query参数解析URL查询参数到ListRequest结构体
	r.GET("node", middleware.BindQueryMiddleware[node_api.ListRequest], app.ListView)

	// GET /node/:id - 获取指定节点详情
	// 绑定URI参数解析URL路径参数（:id）到IDRequest结构体
	r.GET("node/:id", middleware.BindUriMiddleware[models.IDRequest], app.DetailView)

	// PUT /node - 更新节点信息
	// 绑定JSON参数解析请求体JSON数据到UpdateRequest结构体
	r.PUT("node", middleware.BindJsonMiddleware[node_api.UpdateRequest], app.UpdateView)

	// GET /node/options - 获取节点选项
	r.GET("node/options", app.OptionsView)

	// DELETE /node/:id - 删除指定节点
	// 绑定URI参数解析URL路径参数（:id）到IDRequest结构体
	r.DELETE("node/:id", middleware.BindUriMiddleware[models.IDRequest], app.RemoveView)
}
