package host_api

// File: honey_server/api/host_api/remove.go
// Description: 主机删除API接口

import (
	"fmt"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// RemoveView 处理主机批量删除请求
func (HostApi) RemoveView(c *gin.Context) {
	// 获取批量删除请求的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)
	// 获取请求上下文的日志实例
	log := middleware.GetLog(c)

	// 调用通用删除服务执行主机删除操作
	log.WithFields(map[string]interface{}{
		"host_ids":    cr.IdList,
		"total_count": len(cr.IdList),
	}).Info("host deletion request received") // 收到主机删除请求

	successCount, err := common_service.Remove(
		models.HostModel{},
		common_service.RemoveRequest{
			IDList: cr.IdList,
			Log:    log,
			Msg:    "主机",
		},
	)

	// 处理删除操作异常
	if err != nil {
		log.WithFields(map[string]interface{}{
			"host_ids": cr.IdList,
			"error":    err,
		}).Error("failed to delete hosts") // 删除主机失败
		msg := fmt.Sprintf("删除主机失败 %s", err)
		response.FailWithMsg(msg, c)
		return
	}
	log.WithFields(map[string]interface{}{
		"host_ids":        cr.IdList,
		"total_requested": len(cr.IdList),
		"success_count":   successCount,
	}).Info("hosts deletion completed successfully") // 主机删除完成

	// 返回删除成功结果，包含总数与成功数
	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", len(cr.IdList), successCount)
	response.OkWithMsg(msg, c)
}
