package mirror_cloud_api

// File: image_server/api/mirror_cloud_api/image_list.go
// Description: 镜像列表查询API接口

import (
	"ThreatTrapMatrix/apps/image_server/internal/middleware"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/service/common_service"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ImageListRequest 镜像列表查询请求参数结构体
type ImageListRequest struct {
	models.PageInfo // 嵌入分页信息结构体（页码、每页条数等）
}

// ImageListView 镜像列表查询接口处理函数
func (MirrorCloudApi) ImageListView(c *gin.Context) {
	// 获取并绑定列表查询请求参数
	cr := middleware.GetBind[ImageListRequest](c)
	// 调用通用查询服务获取镜像列表及总数
	list, count, _ := common_service.QueryList(models.ImageModel{},
		common_service.QueryListRequest{
			Likes:    []string{"title", "image_name"}, // 支持按别名和镜像名称模糊搜索
			PageInfo: cr.PageInfo,                     // 分页参数
			Sort:     "created_at desc",               // 按创建时间降序排序
		})
	// 返回列表数据及分页信息
	response.OkWithList(list, count, c)
}
