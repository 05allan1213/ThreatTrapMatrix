package user_api

// File: honey_server/api/user_api/remove.go
// Description: 用户批量删除API接口

import (
	middleware2 "ThreatTrapMatrix/apps/honey_server/internal/middleware"
	"ThreatTrapMatrix/apps/honey_server/internal/models"
	"ThreatTrapMatrix/apps/honey_server/internal/service/common_service"
	"ThreatTrapMatrix/apps/honey_server/internal/utils/response"
	"fmt"

	"github.com/gin-gonic/gin"
)

// UserRemoveRequest 批量删除用户的请求参数结构体
type UserRemoveRequest struct {
	IDList []uint `json:"idList"` // 需要删除的用户ID列表
}

// UserRemoveView 批量删除用户接口处理函数
func (UserApi) UserRemoveView(c *gin.Context) {
	// 获取绑定的批量删除请求参数
	cr := middleware2.GetBind[UserRemoveRequest](c)
	// 获取上下文日志实例
	log := middleware2.GetLog(c)
	// 调用通用服务进行批量删除
	successCount, err := common_service.Remove(models.UserModel{}, common_service.RemoveRequest{
		IDList: cr.IDList,
		Log:    log,
		Msg:    "用户",
	})
	if err != nil {
		msg := fmt.Sprintf("删除用户失败 %s", err)
		response.FailWithMsg(msg, c)
		return
	}
	// 构建删除结果消息
	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", len(cr.IDList), successCount)
	// 返回删除结果响应
	response.OkWithMsg(msg, c)
}
