package routers

// File:honey_server/routers/index_routers.go
// Description: 首页路由注册

import (
	"honey_server/internal/api"

	"github.com/gin-gonic/gin"
)

func IndexRouters(r *gin.RouterGroup) {
	var app = api.App.IndexApi
	// GET /index/count: 获取首页统计数据
	r.GET("index/count", app.IndexCountView)
}
