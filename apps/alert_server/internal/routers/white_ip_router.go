package routers

// File: alert_server/routers/white_ip_router.go
// Description: 白名单IP管理路由配置模块，负责注册白名单IP的查询、创建、更新、批量删除接口，绑定请求参数处理中间件

import (
	"alert_server/internal/api"
	"alert_server/internal/api/white_ip_api"
	"alert_server/internal/middleware"
	"alert_server/internal/models"

	"github.com/gin-gonic/gin"
)

// WhiteIpRouter 注册白名单IP管理相关API路由，关联请求参数绑定中间件与对应业务处理接口
func WhiteIpRouter(r *gin.RouterGroup) {
	app := api.App.WhiteIPApi // 白名单IP业务接口实例

	// GET /white_ip: 白名单IP列表查询接口
	// 绑定分页查询参数中间件
	r.GET("white_ip", middleware.BindQueryMiddleware[models.PageInfo], app.ListView)
	// POST /white_ip: 白名单IP创建接口
	// 绑定JSON请求参数中间件
	r.POST("white_ip", middleware.BindJsonMiddleware[white_ip_api.CreateRequest], app.CreateView)
	// PUT /white_ip: 白名单IP更新接口
	// 绑定JSON请求参数中间件
	r.PUT("white_ip", middleware.BindJsonMiddleware[white_ip_api.UpdateRequest], app.UpdateView)
	// DELETE /white_ip: 白名单IP批量删除接口
	// 绑定JSON请求参数中间件
	r.DELETE("white_ip", middleware.BindJsonMiddleware[models.IDListRequest], app.RemoveView)
	// DELETE /white_ip/ip: 白名单IP根据IP删除接口
	// 绑定JSON请求参数中间件
	r.DELETE("white_ip/ip", middleware.BindJsonMiddleware[white_ip_api.RemoveByIpRequest], app.RemoveByIpView)
}
