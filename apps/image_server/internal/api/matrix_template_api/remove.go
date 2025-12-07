package matrix_template_api

// File: image_server/api/matrix_template_api/remove.go
// Description: 矩阵模板批量删除API接口

import (
	"fmt"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/service/common_service"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// Remove 矩阵模板批量删除接口处理函数
func (MatrixTemplateApi) Remove(c *gin.Context) {
	// 获取并绑定批量删除的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)
	// 获取请求上下文日志实例
	log := middleware.GetLog(c)

	log.WithFields(map[string]interface{}{
		"matrix_template_ids": cr.IdList,
		"total_count":         len(cr.IdList),
	}).Info("matrix template deletion request received") // 收到矩阵模板批量删除请求

	// 调用通用服务进行批量删除
	successCount, err := common_service.Remove(
		models.MatrixTemplateModel{},
		common_service.RemoveRequest{
			IDList: cr.IdList,
			Log:    log,
			Msg:    "矩阵模板",
		},
	)

	// 处理删除失败情况
	if err != nil {
		log.WithFields(map[string]interface{}{
			"matrix_template_ids": cr.IdList,
			"error":               err,
		}).Error("failed to delete matrix templates") // 矩阵模板删除失败
		msg := fmt.Sprintf("删除矩阵模板失败 %s", err)
		response.FailWithMsg(msg, c)
		return
	}

	log.WithFields(map[string]interface{}{
		"matrix_template_ids": cr.IdList,
		"total_requested":     len(cr.IdList),
		"success_count":       successCount,
	}).Info("matrix templates deletion completed") // 矩阵模板删除完成

	// 构建成功提示信息（包含请求总数和成功数）
	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", len(cr.IdList), successCount)
	response.OkWithMsg(msg, c)
}
