package index_api

// File: alert_server/api/index_api/service_agg.go
// Description: 虚拟服务攻击聚合统计API接口,为首页提供虚拟服务被攻击次数排行展示

import (
	"alert_server/internal/es_models"
	"alert_server/internal/global"
	"alert_server/internal/utils/response"
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

// ServiceAggResponse 服务攻击聚合统计响应结构体，返回单个服务的核心统计信息
type ServiceAggResponse struct {
	ServiceName   string `json:"serviceName"`   // 被攻击的服务名称（关联虚拟服务名称）
	AttackCount   int    `json:"attackCount"`   // 该服务的总被攻击次数（攻击频次）
	NewAttackDate string `json:"newAttackDate"` // 该服务的最新被攻击时间
}

// ServiceAggType ES服务攻击聚合结果解析结构体，对应ES terms聚合+子聚合返回的JSON格式
type ServiceAggType struct {
	DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"` // 聚合结果误差上限
	SumOtherDocCount        int `json:"sum_other_doc_count"`         // 未返回的聚合桶数量
	Buckets                 []struct {
		Key      string `json:"key"`       // 聚合维度值（即服务名称，serviceName.keyword字段值）
		DocCount int    `json:"doc_count"` // 该服务的总被攻击次数（聚合计数）
		MaxDate  struct { // 子聚合：服务最新被攻击时间（timestamp字段最大值）
			Value         float64 `json:"value"`           // 时间戳（毫秒级）
			ValueAsString string  `json:"value_as_string"` // 格式化后的最新被攻击时间字符串
		} `json:"maxDate"`
	} `json:"buckets"` // 服务攻击聚合桶列表（按被攻击频次降序排列）
}

// ServiceAggView 服务攻击Top5聚合统计接口
func (IndexApi) ServiceAggView(c *gin.Context) {
	// 构建ES聚合查询：主聚合按服务名称（serviceName.keyword）分组，取Top5；子聚合获取每个服务的最新被攻击时间
	agg := elastic.NewTermsAggregation().
		Field("serviceName.keyword"). // 主聚合维度：服务名称
		Size(5). // 限制返回前5条高频被攻击服务
		SubAggregation("maxDate", elastic.NewMaxAggregation().Field("timestamp")) // 子聚合：最新被攻击时间

	// 构建查询条件：排除serviceName为空的告警记录
	query := elastic.NewBoolQuery()
	query.MustNot(elastic.NewTermQuery("serviceName.keyword", "")) // 过滤服务名称为空的记录

	// 执行ES聚合查询：仅返回聚合结果
	res, err := global.ES.Search(es_models.AlertModel{}.Index()).
		Query(query). // 应用过滤条件（排除空服务名）
		Aggregation("agg", agg). // 绑定聚合查询配置
		Size(0). // 聚合查询无需返回文档
		Do(context.Background())
	if err != nil {
		logrus.Errorf("告警服务攻击聚合查询失败 %s", err)
		response.FailWithMsg("告警查询失败", c)
		return
	}

	// 解析ES聚合结果到结构体
	var aggType SrcIpAggType
	err = json.Unmarshal(res.Aggregations["agg"], &aggType)
	if err != nil {
		logrus.Errorf("服务攻击聚合结果json解析失败 %s %s", err, res.Aggregations["agg"])
		response.FailWithMsg("数据解析失败", c)
		return
	}

	// 构造标准化响应数据列表
	var list = make([]ServiceAggResponse, 0, len(aggType.Buckets))
	for _, bucket := range aggType.Buckets {
		list = append(list, ServiceAggResponse{
			ServiceName:   bucket.Key,                   // 被攻击的服务名称
			AttackCount:   bucket.DocCount,              // 该服务的总被攻击次数
			NewAttackDate: bucket.MaxDate.ValueAsString, // 该服务的最新被攻击时间
		})
	}

	// 返回Top5被攻击服务聚合统计结果
	response.OkWithData(list, c)
}
