package routers

// File: honey_server/routers/user_routers.go
// Description: 用户模块路由定义，注册用户相关API接口

import (
	"ThreatTrapMatrix/apps/honey_server/internal/api"
	user_api2 "ThreatTrapMatrix/apps/honey_server/internal/api/user_api"
	middleware2 "ThreatTrapMatrix/apps/honey_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// UserRouters 注册用户相关路由
func UserRouters(r *gin.RouterGroup) {
	// POST /login - 用户登录接口
	// 使用JSON参数绑定中间件解析登录请求参数
	app := api.App.UserApi
	r.POST("login", middleware2.BindJsonMiddleware[user_api2.LoginRequest], app.LoginView)
	// POST /users - 创建用户接口
	// 使用JSON参数绑定中间件解析创建用户请求参数
	r.POST("users", middleware2.AdminMiddleware, middleware2.BindJsonMiddleware[user_api2.CreateRequest], app.CreateView)
	// GET /users - 用户列表查询接口
	// 使用Query参数绑定中间件解析用户列表查询请求参数
	r.GET("users", middleware2.BindQueryMiddleware[user_api2.UserListRequest], app.UserListView)
	// POST /logout - 用户注销接口
	r.POST("logout", app.UserLogoutView)
	// DELETE /users - 用户删除接口
	// 使用JSON参数绑定中间件解析删除用户请求参数
	r.DELETE("users", middleware2.BindJsonMiddleware[user_api2.UserRemoveRequest], app.UserRemoveView)
	// GET /users/info - 用户信息查询接口
	r.GET("users/info", middleware2.AuthMiddleware, app.UserInfoView)
}
