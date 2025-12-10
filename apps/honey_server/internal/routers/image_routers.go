package routers

// File: honey_server/routers/image_routers.go
// Description: 图片路由注册

import (
	"honey_server/internal/api"

	"github.com/gin-gonic/gin"
)

func ImageRouters(r *gin.RouterGroup) {
	var app = api.App.ImageApi
	// POST /image/upload: 图片上传接口
	r.POST("image/upload", app.ImageUploadView)
}
