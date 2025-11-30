package routers

// File: honey_server/routers/enter.go
// Description: 路由模块，负责初始化Gin引擎、注册API路由并启动HTTP服务

import (
	"honey_server/internal/global"
	middleware2 "honey_server/internal/middleware"

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
	g := r.Group("honey_server")
	g.Use(middleware2.LogMiddleware, middleware2.AuthMiddleware) // 系统内部必须登录才能继续使用

	// 路由注册
	UserRouters(g)        // 注册用户相关路由
	CaptchaRouters(g)     // 注册验证码相关路由
	LogRouters(g)         // 注册日志相关路由
	NodeRouters(g)        // 节点相关路由
	NodeNetworkRouters(g) // 节点网卡相关路由
	NetRouters(g)         // 网络相关路由

	// 获取HTTP服务监听地址
	webAddr := system.WebAddr
	logrus.Infof("web addr run %s", webAddr)

	// 启动HTTP服务
	r.Run(webAddr)
}
