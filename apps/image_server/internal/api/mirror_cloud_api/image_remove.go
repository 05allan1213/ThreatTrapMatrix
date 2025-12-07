package mirror_cloud_api

// File: image_server/api/mirror_cloud_api/image_remove.go
// Description: 镜像删除API接口

import (
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ImageRemoveView 镜像删除接口处理函数
func (MirrorCloudApi) ImageRemoveView(c *gin.Context) {
	// 获取镜像ID请求参数
	cr := middleware.GetBind[models.IDRequest](c)
	// 获取日志实例
	log := middleware.GetLog(c)

	log.WithFields(map[string]interface{}{
		"image_id": cr.ID,
	}).Info("received image deletion request") // 收到镜像删除请求

	var model models.ImageModel
	// 查询镜像信息并预加载关联的虚拟服务列表
	if err := global.DB.Preload("ServiceList").Take(&model, cr.ID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"image_id": cr.ID,
			"error":    err,
		}).Warn("failed to query image (may not exist)") // 查询镜像失败（可能不存在）
		response.FailWithMsg("镜像不存在", c)
		return
	}
	// 校验镜像是否关联虚拟服务（存在则禁止删除）
	if len(model.ServiceList) > 0 {
		log.WithFields(map[string]interface{}{
			"image_id":              model.ID,
			"image_name":            model.ImageName,
			"related_service_count": len(model.ServiceList),
		}).Warn("image has related virtual services, deletion rejected") // 镜像存在关联的虚拟服务，拒绝删除
		response.FailWithMsg("镜像存在虚拟服务，请先删除关联的虚拟服务", c)
		return
	}

	// 记录镜像删除日志
	log.Infof("删除镜像 %#v", model)

	// 删除数据库中的镜像记录
	if err := global.DB.Delete(&model).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"image_id":   model.ID,
			"image_name": model.ImageName,
			"error":      err,
		}).Error("failed to delete image record from database") // 删除数据库中的镜像记录失败
		response.FailWithMsg("镜像删除失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"image_id":   model.ID,
		"image_name": model.ImageName,
		"image_tag":  model.Tag,
	}).Info("image record deleted successfully") // 镜像记录删除成功

	response.OkWithMsg("删除成功", c)
}
