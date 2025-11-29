package mirror_cloud_api

// File: image_server/api/mirror_cloud_api/image_detail.go
// Description: 镜像文件详情API接口

import (
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ImageDetailView 镜像详情查询接口
func (MirrorCloudApi) ImageDetailView(c *gin.Context) {
	// 获取并绑定镜像ID请求参数
	cr := middleware.GetBind[models.IDRequest](c)
	var model models.ImageModel
	// 从数据库查询指定ID的镜像信息
	err := global.DB.Take(&model, cr.ID).Error
	if err != nil {
		response.FailWithMsg("镜像不存在", c)
		return
	}

	// 返回镜像详情数据
	response.OkWithData(model, c)
}
