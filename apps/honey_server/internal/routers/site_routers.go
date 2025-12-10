package routers

// File: honey_server/routers/site_routers.go
// Description: 站点配置路由

import (
	"honey_server/internal/api"
	"honey_server/internal/config"
	"honey_server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SiteRouters 站点配置路由
func SiteRouters(r *gin.RouterGroup) {
	var app = api.App.SiteApi
	// PUT /site: 站点配置更新接口
	r.PUT("site", middleware.BindJsonMiddleware[config.Site], app.SiteUpdateView)
	// GET /site: 站点配置接口
	r.GET("site", app.SiteInfoView)
}
