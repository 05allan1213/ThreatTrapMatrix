package api

// File: matrix_server/api/net_progress.go
// Description: 子网部署进度查询API接口

import (
	"matrix_server/internal/global"
	"matrix_server/internal/middleware"
	"matrix_server/internal/models"
	"matrix_server/internal/service/redis_service/net_progress"
	"matrix_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// NetProgressResponse 子网部署进度查询响应结构体
type NetProgressResponse struct {
	net_progress.NetDeployInfo         // 嵌入子网部署进度基础信息结构体（含总数、完成数、错误数等）
	Progress                   float64 `json:"progress"` // 部署进度百分比（取值范围0-100）
}

// NetProgressView 子网部署进度查询接口处理函数
func (Api) NetProgressView(c *gin.Context) {
	// 绑定并解析请求参数（子网ID）
	cr := middleware.GetBind[models.IDRequest](c)
	// 查询子网基础信息，校验子网是否存在
	var model models.NetModel
	err := global.DB.Take(&model, cr.Id).Error
	if err != nil {
		response.FailWithMsg("子网不存在", c)
		return
	}

	// 从Redis读取子网部署进度信息（忽略读取失败，返回空进度数据）
	progressInfo, _ := net_progress.Get(cr.Id)
	// 组装进度响应数据，计算进度百分比
	data := NetProgressResponse{
		NetDeployInfo: progressInfo,
		Progress:      (float64(progressInfo.CompletedCount) / float64(progressInfo.AllCount)) * 100,
	}
	// 返回进度数据给前端
	response.OkWithData(data, c)
}
