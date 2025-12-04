package routers

// File: alert_server/routers/enter.go
// Description: 路由模块，负责初始化Gin引擎、注册API路由并启动HTTP服务

import (
	"alert_server/internal/global"
	"alert_server/internal/middleware"

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

	// 创建API根路由分组
	g := r.Group("alert_server")
	g.Use(middleware.LogMiddleware, middleware.AuthMiddleware) // 系统内部必须登录才能继续使用

	// 路由注册
	WhiteIpRouter(g) // 白名单ip相关路由
	AlertRouter(g)   // 告警相关路由
	IndexRouter(g)   // 首页相关路由

	// 获取HTTP服务监听地址
	webAddr := system.WebAddr
	logrus.Infof("web addr run %s", webAddr)

	// 启动HTTP服务
	r.Run(webAddr)
}
