package node_api

// File: honey_server/api/node_api/detail.go
// Description: 节点详情API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// DetailView 节点详情查询接口处理函数
func (NodeApi) DetailView(c *gin.Context) {
	// 从请求中绑定并获取ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	// 定义节点模型对象，用于接收数据库查询结果
	var model models.NodeModel

	// 根据ID查询数据库中的节点记录
	err := global.DB.Take(&model, cr.Id).Error
	if err != nil {
		// 查询失败（如ID不存在），返回标准化失败响应
		response.FailWithMsg("节点不存在", c)
		return
	}

	// 查询成功，返回标准化成功响应
	response.OkWithData(model, c)
}
