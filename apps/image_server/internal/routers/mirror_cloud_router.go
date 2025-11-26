package routers

// File: image_server/routers/mirror_cloud_router.go
// Description: 镜像云相关路由

import (
	"ThreatTrapMatrix/apps/image_server/internal/api"

	"github.com/gin-gonic/gin"
)

// MirrorCloudRouter 注册镜像云相关路由
func MirrorCloudRouter(r *gin.RouterGroup) {
	app := api.App.MirrorCloudApi
	// POST /mirror_cloud/see - 镜像文件查看接口
	r.POST("mirror_cloud/see", app.ImageSeeView)
}
