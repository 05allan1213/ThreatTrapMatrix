package vs_api

// File: image_server/api/vs_api/vs_remove.go
// Description: 虚拟服务批量删除API接口实现

import (
	"fmt"
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// VsRemoveView 虚拟服务批量删除接口处理函数
func (VsApi) VsRemoveView(c *gin.Context) {
	log := middleware.GetLog(c)

	// 获取并绑定批量删除的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)

	log.WithFields(map[string]interface{}{
		"service_ids": cr.IdList,
	}).Info("virtual service deletion request received") // 收到虚拟服务批量删除请求

	// 根据ID列表查询对应的虚拟服务记录
	var serviceList []models.ServiceModel
	if err := global.DB.Find(&serviceList, "id in ?", cr.IdList).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"service_ids": cr.IdList,
			"error":       err,
		}).Error("failed to query services for deletion") // 查询虚拟服务失败
		response.FailWithMsg("查询虚拟服务失败", c)
		return
	}

	// 校验是否存在有效服务记录
	if len(serviceList) == 0 {
		log.WithFields(map[string]interface{}{
			"service_ids": cr.IdList,
		}).Warn("no services found for deletion") // 未找到要删除的虚拟服务
		response.FailWithMsg("不存在的虚拟服务", c)
		return
	}

	// 获取服务ID列表
	var serviceIDs []uint
	for _, service := range serviceList {
		serviceIDs = append(serviceIDs, service.ID)
	}

	// 执行批量删除操作
	result := global.DB.Delete(&serviceList)
	successCount := result.RowsAffected // 获取成功删除的记录数
	if err := result.Error; err != nil {
		log.WithFields(map[string]interface{}{
			"service_ids": serviceIDs,
			"error":       err,
		}).Error("failed to delete virtual services") // 批量删除虚拟服务失败
		response.FailWithMsg("删除虚拟服务失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"deleted_service_ids": serviceIDs,
		"success_count":       successCount,
	}).Info("virtual services deleted successfully") // 虚拟服务删除成功

	// 构建成功提示信息并返回
	msg := fmt.Sprintf("删除虚拟服务成功 共%d个", successCount)
	response.OkWithMsg(msg, c)
}
