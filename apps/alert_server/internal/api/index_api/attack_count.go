package index_api

// File: alert_server/api/index_api/attack_count.go
// Description: 告警服务总告警次数统计API接口

import (
	"alert_server/internal/es_models"
	"alert_server/internal/global"
	"alert_server/internal/utils/response"
	"context"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

// AttackCountResponse 总攻击次数统计响应结构体，返回所有告警记录的总数量
type AttackCountResponse struct {
	AttackCount int64 `json:"attackCount"` // 告警总数量（即所有攻击记录的累计次数）
}

// AttackCountView 告警总数量统计接口
func (IndexApi) AttackCountView(c *gin.Context) {
	// 构建ES ValueCount聚合：统计文档总数量，基于_id字段
	agg := elastic.NewValueCountAggregation().Field("_id")

	// 执行ES聚合查询：仅返回聚合结果
	res, err := global.ES.Search(es_models.AlertModel{}.Index()).
		Aggregation("agg", agg). // 绑定总数量统计聚合配置
		Size(0). // 聚合查询无需返回文档
		Do(context.Background())
	if err != nil {
		logrus.Errorf("告警总数量统计查询失败 %s", err)
		response.FailWithMsg("告警查询失败", c)
		return
	}

	// 解析ES聚合结果：获取总数量统计值
	countResponse, ok := res.Aggregations.ValueCount("agg")
	if !ok {
		logrus.Warn("告警总数量聚合结果不存在")
		response.FailWithMsg("不存在的聚合", c)
		return
	}

	// 转换聚合结果为标准化响应格式，返回总攻击次数
	response.OkWithData(AttackCountResponse{
		AttackCount: int64(*countResponse.Value),
	}, c)
}
