package routers

// File: alert_server/routers/index_router.go
// Description: 首页数据路由配置模块

import (
	"alert_server/internal/api"

	"github.com/gin-gonic/gin"
)

// IndexRouter 注册首页相关路由
func IndexRouter(r *gin.RouterGroup) {
	app := api.App.IndexApi // 首页API接口实例

	// GET /index/signature_agg: 首页攻击类型Top5聚合统计接口，返回出现频次最高的5种攻击类型及对应攻击次数
	r.GET("index/signature_agg", app.SignatureAggView)
}
