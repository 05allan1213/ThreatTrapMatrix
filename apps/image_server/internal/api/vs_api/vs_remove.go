package vs_api

// File: image_server/api/vs_api/vs_remove.go
// Description: 虚拟服务批量删除API接口实现

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/middleware"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"
	"fmt"

	"github.com/gin-gonic/gin"
)

// VsRemoveView 虚拟服务批量删除接口处理函数
func (VsApi) VsRemoveView(c *gin.Context) {
	// 获取并绑定批量删除的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)

	// 根据ID列表查询对应的虚拟服务记录
	var serviceList []models.ServiceModel
	global.DB.Find(&serviceList, "id in ?", cr.IdList)

	// 校验是否存在有效服务记录
	if len(serviceList) == 0 {
		response.FailWithMsg("不存在的虚拟服务", c)
		return
	}

	// 执行批量删除操作
	result := global.DB.Delete(&serviceList)
	successCount := result.RowsAffected // 获取成功删除的记录数
	err := result.Error                 // 获取删除操作错误信息

	// 处理删除失败情况
	if err != nil {
		response.FailWithMsg("删除虚拟服务失败", c)
		return
	}

	// 构建成功提示信息并返回
	msg := fmt.Sprintf("删除虚拟服务成功 共%d个", successCount)
	response.OkWithMsg(msg, c)
}
