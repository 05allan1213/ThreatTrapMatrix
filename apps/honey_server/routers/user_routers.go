package routers

// File: honey_server/routers/user_routers.go
// Description: 用户模块路由定义，注册用户相关API接口

import (
	"ThreatTrapMatrix/apps/honey_server/api"
	"ThreatTrapMatrix/apps/honey_server/api/user_api"
	"ThreatTrapMatrix/apps/honey_server/middleware"

	"github.com/gin-gonic/gin"
)

// UserRouters 注册用户相关路由
func UserRouters(r *gin.RouterGroup) {
	// POST /honey_server/login - 用户登录接口
	// 使用JSON参数绑定中间件解析登录请求参数
	app := api.App.UserApi
	r.POST("login", middleware.BindJsonMiddleware[user_api.LoginRequest], app.LoginView)
}
