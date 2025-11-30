package net_api

// File: honey_server/api/net_api/remove.go
// Description: 网络模块删除API接口

import (
	"fmt"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// RemoveView 处理网络批量删除请求，支持多ID删除并返回删除统计结果
func (NetApi) RemoveView(c *gin.Context) {
	// 绑定并获取批量删除的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)
	// 获取上下文绑定的日志实例
	log := middleware.GetLog(c)

	// 调用通用删除服务执行批量删除操作
	successCount, err := common_service.Remove(models.NetModel{}, common_service.RemoveRequest{
		IDList: cr.IdList, // 待删除的网络ID列表
		Log:    log,       // 日志实例，用于记录删除过程
		Msg:    "网络",    // 业务模块名称，用于日志和提示信息
	})

	// 处理删除操作错误
	if err != nil {
		msg := fmt.Sprintf("删除网络失败 %s", err)
		response.FailWithMsg(msg, c)
		return
	}

	// 构造删除成功提示信息，包含总数和成功数
	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", len(cr.IdList), successCount)
	response.OkWithMsg(msg, c)
}
