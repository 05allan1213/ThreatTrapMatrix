package mirror_cloud_api

// File: image_server/api/mirror_cloud_api/image_update.go
// Description: 镜像更新API接口

import (
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ImageUpdateRequest 镜像更新接口请求参数结构体
type ImageUpdateRequest struct {
	ID        uint   `json:"id"`                                      // 镜像ID
	Title     string `json:"title" binding:"required"`                // 镜像别名
	Port      int    `json:"port" binding:"required,min=1,max=65535"` // 镜像运行端口
	Agreement int8   `json:"agreement" binding:"required,oneof=1"`    // 镜像通信协议（1为固定协议类型）
	Status    int8   `json:"status" binding:"required,oneof=1 2"`     // 镜像状态
	Logo      string `json:"logo"`                                    // 镜像logo
	Desc      string `json:"desc"`                                    // 镜像描述
}

// ImageUpdateView 镜像更新接口处理函数
func (MirrorCloudApi) ImageUpdateView(c *gin.Context) {
	// 获取并绑定镜像更新请求参数
	cr := middleware.GetBind[ImageUpdateRequest](c)

	// 查询待更新的镜像是否存在
	var model models.ImageModel
	err := global.DB.Take(&model, cr.ID).Error
	if err != nil {
		response.FailWithMsg("镜像不存在", c)
		return
	}

	// 校验修改后的镜像别名是否与其他镜像重复（排除自身ID）
	var newModel models.ImageModel
	err = global.DB.Take(&newModel, "id <> ? and title = ?", cr.ID, cr.Title).Error
	if err == nil {
		response.FailWithMsg("修改的镜像名称不能重复", c)
		return
	}

	// 更新镜像信息到数据库
	err = global.DB.Model(&model).Updates(models.ImageModel{
		Title:     cr.Title,
		Port:      cr.Port,
		Agreement: cr.Agreement,
		Status:    cr.Status,
		Logo:      cr.Logo,
		Desc:      cr.Desc,
	}).Error
	if err != nil {
		response.FailWithMsg("镜像更新失败", c)
		return
	}

	// 返回更新成功响应
	response.OkWithMsg("镜像修改成功", c)
}
