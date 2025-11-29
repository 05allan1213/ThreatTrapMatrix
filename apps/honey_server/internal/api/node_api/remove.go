package node_api

// File: honey_server/api/node_api/remove.go
// Description: 节点删除API接口实现

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// RemoveView 节点删除接口处理函数
func (NodeApi) RemoveView(c *gin.Context) {
	// 从请求中绑定并获取节点ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	// 查询待删除节点是否存在
	var model models.NodeModel
	err := global.DB.Take(&model, cr.Id).Error
	if err != nil {
		response.FailWithMsg("节点不存在", c)
		return
	}

	// 执行节点删除操作(物理删除)
	err = global.DB.Delete(&model).Error
	if err != nil {
		response.FailWithMsg("节点删除失败", c)
		return
	}

	// 删除成功，返回提示信息
	response.OkWithMsg("节点删除成功", c)
}
