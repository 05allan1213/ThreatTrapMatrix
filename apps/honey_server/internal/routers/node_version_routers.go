package routers

// File: honey_server/routers/node_version_routers.go
// Description: 节点版本管理路由注册

import (
	"honey_server/internal/api"
	"honey_server/internal/middleware"
	"honey_server/internal/models"

	"github.com/gin-gonic/gin"
)

// NodeVersionRouters 注册节点版本管理相关路由
func NodeVersionRouters(r *gin.RouterGroup) {
	// 获取节点版本API实例，统一管理版本相关接口的处理函数
	var app = api.App.NodeVersionApi

	// POST /node_version：节点版本创建接口
	// 功能：接收节点版本信息，创建新的节点版本记录
	r.POST("node_version", app.NodeVersionCreateView)

	// GET /node_version：节点版本分页列表查询接口
	// 功能：查询节点版本列表，支持分页；
	// 中间件：BindQueryMiddleware[models.PageInfo] - 自动绑定URL查询参数到分页结构体，校验参数合法性
	r.GET("node_version", middleware.BindQueryMiddleware[models.PageInfo], app.NodeVersionListView)

	// GET /node_version/download：节点版本文件下载接口
	// 功能：根据节点版本ID下载对应的版本文件；
	// 中间件：BindQueryMiddleware[models.IDRequest] - 自动绑定URL查询参数中的ID到ID请求结构体，校验ID合法性
	r.GET("node_version/download", middleware.BindQueryMiddleware[models.IDRequest], app.NodeVersionDownloadView)

	// GET /node_version/options：节点版本选项接口
	// 功能：返回节点版本选项列表，用于前端下拉框选择
	r.GET("node_version/options", app.NodeVersionOptionsView)

	// DELETE /node_version/:id：节点版本删除接口
	// 功能：根据URI中的ID删除指定的节点版本记录及关联文件；
	// 中间件：BindUriMiddleware[models.IDRequest] - 自动绑定URI路径参数（:id）到ID请求结构体，校验ID合法性
	r.DELETE("node_version/:id", middleware.BindUriMiddleware[models.IDRequest], app.NodeVersionRemoveView)
}
