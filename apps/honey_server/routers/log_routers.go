package routers

// File: honey_server/routers/log_routers.go
// Description: 日志模块路由配置，定义日志相关接口的路由规则及中间件绑定

import (
	"ThreatTrapMatrix/apps/honey_server/api"
	"ThreatTrapMatrix/apps/honey_server/api/log_api"
	"ThreatTrapMatrix/apps/honey_server/middleware"
	"ThreatTrapMatrix/apps/honey_server/models"

	"github.com/gin-gonic/gin"
)

// LogRouters 配置日志模块的路由规则
// r: 日志模块所属的路由组实例
func LogRouters(r *gin.RouterGroup) {
	// 获取日志API接口实例
	app := api.App.LogApi
	// GET /logs: 日志列表查询接口，需管理员权限，绑定日志列表查询参数
	r.GET("logs", middleware.AdminMiddleware, middleware.BindQueryMiddleware[log_api.LogListRequest], app.LogListView)
	// DELETE /logs: 日志批量删除接口，需管理员权限，绑定ID列表参数
	r.DELETE("logs", middleware.AdminMiddleware, middleware.BindJsonMiddleware[models.IDListRequest], app.RemoveView)
}
