package routers

// File: image_server/routers/mirror_cloud_router.go
// Description: 镜像云相关路由

import (
	"ThreatTrapMatrix/apps/image_server/internal/api"
	"ThreatTrapMatrix/apps/image_server/internal/api/mirror_cloud_api"
	"ThreatTrapMatrix/apps/image_server/internal/middleware"
	"ThreatTrapMatrix/apps/image_server/internal/models"

	"github.com/gin-gonic/gin"
)

// MirrorCloudRouter 注册镜像云相关路由
func MirrorCloudRouter(r *gin.RouterGroup) {
	app := api.App.MirrorCloudApi
	// POST /mirror_cloud/see - 镜像文件查看接口
	r.POST("mirror_cloud/see", app.ImageSeeView)
	// POST /mirror_cloud - 镜像文件创建接口
	// 使用JSON参数绑定中间件解析创建镜像请求参数
	r.POST("mirror_cloud", middleware.BindJsonMiddleware[mirror_cloud_api.ImageCreateRequest], app.ImageCreateView)
	// GET /mirror_cloud - 镜像文件列表接口
	// 使用Query参数绑定中间件解析列表请求参数
	r.GET("mirror_cloud", middleware.BindQueryMiddleware[mirror_cloud_api.ImageListRequest], app.ImageListView)
	// GET /mirror_cloud/:id - 镜像文件详情接口
	// 使用URI参数绑定中间件解析详情请求参数
	r.GET("mirror_cloud/:id", middleware.BindUriMiddleware[models.IDRequest], app.ImageDetailView)
	// PUT /mirror_cloud - 镜像文件更新接口
	// 使用JSON参数绑定中间件解析更新镜像请求参数
	r.PUT("mirror_cloud", middleware.BindJsonMiddleware[mirror_cloud_api.ImageUpdateRequest], app.ImageUpdateView)
	// DELETE /mirror_cloud/:id - 镜像文件删除接口
	// 使用URI参数绑定中间件解析删除镜像请求参数
	r.DELETE("mirror_cloud/:id", middleware.BindUriMiddleware[models.IDRequest], app.ImageRemoveView)
	// GET /mirror_cloud/options - 镜像文件选项列表接口
	r.GET("mirror_cloud/options", app.ImageOptionsListView)
}
