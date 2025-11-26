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
	// POST /login - 用户登录接口
	// 使用JSON参数绑定中间件解析登录请求参数
	app := api.App.UserApi
	r.POST("login", middleware.BindJsonMiddleware[user_api.LoginRequest], app.LoginView)
	// POST /users - 创建用户接口
	// 使用JSON参数绑定中间件解析创建用户请求参数
	r.POST("users", middleware.AdminMiddleware, middleware.BindJsonMiddleware[user_api.CreateRequest], app.CreateView)
	// GET /users - 用户列表查询接口
	// 使用Query参数绑定中间件解析用户列表查询请求参数
	r.GET("users", middleware.BindQueryMiddleware[user_api.UserListRequest], app.UserListView)
	// POST /logout - 用户注销接口
	r.POST("logout", app.UserLogoutView)
	// DELETE /users - 用户删除接口
	// 使用JSON参数绑定中间件解析删除用户请求参数
	r.DELETE("users", middleware.BindJsonMiddleware[user_api.UserRemoveRequest], app.UserRemoveView)
	// GET /users/info - 用户信息查询接口
	r.GET("users/info", middleware.AuthMiddleware, app.UserInfoView)
}
