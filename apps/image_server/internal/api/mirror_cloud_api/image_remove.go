package mirror_cloud_api

// File: image_server/api/mirror_cloud_api/image_remove.go
// Description: 镜像删除API接口

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/middleware"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ImageRemoveView 镜像删除接口处理函数
func (MirrorCloudApi) ImageRemoveView(c *gin.Context) {
	// 获取镜像ID请求参数
	cr := middleware.GetBind[models.IDRequest](c)
	// 获取日志实例
	log := middleware.GetLog(c)
	var model models.ImageModel
	// 查询镜像信息并预加载关联的虚拟服务列表
	err := global.DB.Preload("ServiceList").Take(&model, cr.ID).Error
	if err != nil {
		response.FailWithMsg("镜像不存在", c)
		return
	}
	// 校验镜像是否关联虚拟服务（存在则禁止删除）
	if len(model.ServiceList) > 0 {
		response.FailWithMsg("镜像存在虚拟服务，请先删除关联的虚拟服务", c)
		return
	}

	// 记录镜像删除日志
	log.Infof("删除镜像 %#v", model)

	// 删除数据库中的镜像记录
	err = global.DB.Delete(&model).Error
	if err != nil {
		log.Errorf("删除镜像失败 %s", err)
		response.FailWithMsg("镜像删除失败", c)
		return
	}
	response.OkWithMsg("删除成功", c)
}
