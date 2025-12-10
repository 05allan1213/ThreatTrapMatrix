package routers

import (
	"image_server/internal/api"

	"github.com/gin-gonic/gin"
)

func IndexRouter(r *gin.RouterGroup) {
	app := api.App.IndexApi
	// GET /index/count: 获取首页统计数据
	r.GET("index/count", app.IndexCountView)
}
