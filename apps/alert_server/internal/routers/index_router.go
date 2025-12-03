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

	// GET /index/signature_agg: 攻击类型Top5聚合统计接口，返回出现频次最高的5种攻击类型及对应攻击次数
	r.GET("index/signature_agg", app.SignatureAggView)
	// GET /index/src_ip_agg: 源ipTop5聚合统计接口，返回出现频次最高的5个源IP及对应攻击次数
	r.GET("index/src_ip_agg", app.SrcIpAggView)
	// GET /index/service_agg: 虚拟服务Top5聚合统计接口，返回出现频次最高的5种虚拟服务及对应攻击次数
	r.GET("index/service_agg", app.ServiceAggView)
	// GET /index/date_agg: 时间聚合接口，返回指定时间段内攻击次数
	r.GET("index/date_agg", app.DateAggView)
	// GET /index/attack_count: 告警服务总告警次数接口，返回告警总次数
	r.GET("index/attack_count", app.AttackCountView)
}
