package matrix_template_api

// File: image_server/api/matrix_template_api/remove.go
// Description: 矩阵模板批量删除API接口

import (
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/service/common_service"
	"image_server/internal/utils/response"
	"fmt"

	"github.com/gin-gonic/gin"
)

// Remove 矩阵模板批量删除接口处理函数
func (MatrixTemplateApi) Remove(c *gin.Context) {
	// 获取并绑定批量删除的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)
	// 获取请求上下文日志实例
	log := middleware.GetLog(c)

	// 调用公共删除服务执行批量删除
	successCount, err := common_service.Remove(models.MatrixTemplateModel{}, common_service.RemoveRequest{
		IDList: cr.IdList,  // 待删除的模板ID列表
		Log:    log,        // 日志实例
		Msg:    "矩阵模板", // 业务类型标识（用于日志记录）
	})

	// 处理删除失败情况
	if err != nil {
		msg := fmt.Sprintf("删除矩阵模板失败 %s", err)
		response.FailWithMsg(msg, c)
		return
	}

	// 构建成功提示信息（包含请求总数和成功数）
	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", len(cr.IdList), successCount)
	response.OkWithMsg(msg, c)
}
