package routers

// File: matrix_server/routers/enter.go
// Description: 路由模块，负责初始化Gin引擎、注册API路由并启动HTTP服务

import (
	"matrix_server/internal/api"
	"matrix_server/internal/global"
	"matrix_server/internal/middleware"
	"matrix_server/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Run 初始化路由引擎并启动HTTP服务
func Run() {
	// 获取系统配置信息
	system := global.Config.System
	// 设置Gin运行模式（debug/release/test）
	gin.SetMode(system.Mode)

	// 创建默认Gin引擎
	r := gin.Default()
	// 创建静态路由
	r.Static("uploads", "uploads")
	// 创建API根路由分组
	g := r.Group("matrix_server")
	g.Use(middleware.LogMiddleware, middleware.AuthMiddleware) // 系统内部必须登录才能继续使用
	// GET /net/ip_list : 获取网络IP列表
	g.GET("net/ip_list", middleware.BindQueryMiddleware[api.NetIpListRequest], api.App.NetIpListView)
	// POST /deploy : 批量部署
	g.POST("deploy", middleware.BindJsonMiddleware[api.DeployRequest], api.App.DeployView)
	// PUT /deploy : 批量部署更新
	g.PUT("deploy", middleware.BindJsonMiddleware[api.DeployRequest], api.App.UpdateDeployView)
	// DELETE /deploy : 批量部署删除
	g.DELETE("deploy", middleware.BindJsonMiddleware[api.RemoveDeployRequest], api.App.RemoveDeployView)
	// GET /deploy/progress/:id : 获取部署进度
	g.GET("deploy/progress/:id", middleware.BindUriMiddleware[models.IDRequest], api.App.NetProgressView)

	// 获取HTTP服务监听地址
	webAddr := system.WebAddr
	logrus.Infof("web addr run %s", webAddr)

	// 启动HTTP服务
	r.Run(webAddr)
}
