package routers

// File: alert_server/routers/alert_routers.go
// Description: 告警相关路由模块

import (
	"alert_server/internal/api"
	"alert_server/internal/api/alert_api"
	"alert_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func AlertRouter(r *gin.RouterGroup) {
	app := api.App.AlertApi
	// GET: /alert - 获取告警列表
	// 绑定Query查询参数到ListRequest结构体，并调用AlertApi的ListView方法
	r.GET("alert", middleware.BindQueryMiddleware[alert_api.ListRequest], app.ListView)
	// GET: /alert/src_ip_agg - 获取告警源IP聚合列表
	// 绑定Query查询参数到SrcIpAggRequest结构体，并调用AlertApi的SrcIpAggView方法
	r.GET("src_ip_agg", middleware.BindQueryMiddleware[alert_api.SrcIpAggRequest], app.SrcIpAggView)
	// DELETE: /alert - 删除告警记录
	// 绑定JSON请求参数到RemoveRequest结构体，并调用AlertApi的RemoveView方法
	r.DELETE("alert", middleware.BindJsonMiddleware[alert_api.RemoveRequest], app.RemoveView)
}
