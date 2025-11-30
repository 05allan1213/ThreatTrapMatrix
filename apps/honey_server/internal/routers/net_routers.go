package routers

// File: honey_server/routers/net_routers.go
// Description: 网络模块路由配置，定义网络相关接口的路由规则及中间件绑定

import (
	"honey_server/internal/api"
	"honey_server/internal/api/net_api"
	"honey_server/internal/middleware"
	"honey_server/internal/models"

	"github.com/gin-gonic/gin"
)

// NetRouters 注册网络模块相关路由
func NetRouters(r *gin.RouterGroup) {
	// 获取网络API实例
	var app = api.App.NetApi
	// GET /net - 获取网络列表
	// 绑定Query参数结构体,解析URL查询参数到ListRequest结构体
	r.GET("net", middleware.BindQueryMiddleware[net_api.ListRequest], app.ListView)
	// GET /net/options - 获取网络列表选项
	r.GET("net/options", app.OptionsView)
	// GET /net/:id - 获取指定网络详情
	// 绑定URI参数结构体,解析URL路径参数到IDRequest结构体
	r.GET("net/:id", middleware.BindUriMiddleware[models.IDRequest], app.DetailView)
	// PUT /net - 更新网络信息
	// 绑定JSON参数结构体,解析请求体JSON数据到UpdateRequest结构体
	r.PUT("net", middleware.BindJsonMiddleware[net_api.UpdateRequest], app.UpdateView)
	// DELETE /net - 删除网络
	// 绑定JSON参数结构体,解析请求体JSON数据到IDListRequest结构体
	r.DELETE("net", middleware.BindJsonMiddleware[models.IDListRequest], app.RemoveView)
}
