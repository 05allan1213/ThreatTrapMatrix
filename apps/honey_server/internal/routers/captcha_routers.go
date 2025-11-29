package routers

// File: honey_server/routers/captcha_routers.go
// Description: 验证码模块路由定义，注册验证码相关API接口

import (
	"honey_server/internal/api"

	"github.com/gin-gonic/gin"
)

// CaptchaRouters 注册验证码相关路由
func CaptchaRouters(r *gin.RouterGroup) {
	// 获取验证码API实例
	app := api.App.CaptchaApi

	// GET /honey_server/captcha - 生成图片验证码接口
	r.GET("captcha", app.GenerateView)
}
