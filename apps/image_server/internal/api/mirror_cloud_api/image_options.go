package mirror_cloud_api

// File: image_server/api/mirror_cloud_api/image_options.go
// Description: 镜像选项列表API接口

import (
	"image_server/internal/global"
	"image_server/internal/models"
	"image_server/internal/utils/response"
	"fmt"

	"github.com/gin-gonic/gin"
)

// ImageOptionsListResponse 镜像选项列表响应结构体
type ImageOptionsListResponse struct {
	Label   string `json:"label"`   // 选项展示文本（镜像别名+端口）
	Value   uint   `json:"value"`   // 选项值（镜像ID）
	Disable bool   `json:"disable"` // 是否禁用选项（镜像状态为禁用时生效）
}

// ImageOptionsListView 获取镜像选项列表接口
func (MirrorCloudApi) ImageOptionsListView(c *gin.Context) {
	// 查询所有镜像记录
	var list []models.ImageModel
	global.DB.Find(&list)
	// 组装镜像选项数据
	var options []ImageOptionsListResponse
	for _, model := range list {
		item := ImageOptionsListResponse{
			Label: fmt.Sprintf("%s/%d", model.Title, model.Port), // 拼接展示文本（别名/端口）
			Value: model.ID,
		}
		// 镜像状态为禁用（2）时设置选项禁用
		if model.Status == 2 {
			item.Disable = true
		}
		options = append(options, item)
	}
	// 返回镜像选项列表数据
	response.OkWithData(options, c)
}
