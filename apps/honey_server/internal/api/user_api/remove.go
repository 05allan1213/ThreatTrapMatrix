package user_api

// File: honey_server/api/user_api/remove.go
// Description: 用户批量删除API接口

import (
	"fmt"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// UserRemoveRequest 批量删除用户的请求参数结构体
type UserRemoveRequest struct {
	IDList []uint `json:"idList"` // 需要删除的用户ID列表
}

// UserRemoveView 批量删除用户接口处理函数
func (UserApi) UserRemoveView(c *gin.Context) {
	// 获取绑定的批量删除请求参数
	cr := middleware.GetBind[UserRemoveRequest](c)
	// 获取上下文日志实例
	log := middleware.GetLog(c)
	// 调用通用服务进行批量删除
	log.WithFields(map[string]interface{}{
		"user_ids":    cr.IDList,
		"total_count": len(cr.IDList),
	}).Info("user deletion request received") // 收到删除用户请求

	successCount, err := common_service.Remove(
		models.UserModel{},
		common_service.RemoveRequest{
			IDList: cr.IDList,
			Log:    log,
			Msg:    "用户",
		},
	)

	if err != nil {
		log.WithFields(map[string]interface{}{
			"user_ids": cr.IDList,
			"error":    err,
		}).Error("failed to delete users") // 删除用户失败
		msg := fmt.Sprintf("删除用户失败 %s", err)
		response.FailWithMsg(msg, c)
		return
	}

	log.WithFields(map[string]interface{}{
		"user_ids":        cr.IDList,
		"total_requested": len(cr.IDList),
		"success_count":   successCount,
	}).Info("user deletion completed successfully") // 用户删除完成
	// 构建删除结果消息
	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", len(cr.IDList), successCount)
	// 返回删除结果响应
	response.OkWithMsg(msg, c)
}
