package index_api

// File: alert_server/api/index_api/src_ip_agg.go
// Description: 攻击源IP聚合统计API接口层，为首页提供攻击源IP排行展示

import (
	"alert_server/internal/core"
	"alert_server/internal/es_models"
	"alert_server/internal/global"
	"alert_server/internal/utils/response"
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

// SrcIpAggResponse 攻击源IP聚合统计响应结构体，返回单个攻击源IP的核心统计信息
type SrcIpAggResponse struct {
	SrcIp         string `json:"srcIp"`         // 攻击源IP地址
	Addr          string `json:"addr"`          // 攻击源IP归属地（通过IP解析工具获取）
	AttackCount   int    `json:"attackCount"`   // 该源IP的总攻击次数（攻击频次）
	NewAttackDate string `json:"newAttackDate"` // 该源IP的最新攻击时间（格式化字符串）
}

// SrcIpAggType ES攻击源IP聚合结果解析结构体，对应ES terms聚合+子聚合返回的JSON格式
type SrcIpAggType struct {
	DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"` // 聚合结果误差上限
	SumOtherDocCount        int `json:"sum_other_doc_count"`         // 未返回的聚合桶数量
	Buckets                 []struct {
		Key      string `json:"key"`       // 聚合维度值（即攻击源IP地址）
		DocCount int    `json:"doc_count"` // 该源IP的总攻击次数（聚合计数）
		MaxDate  struct { // 子聚合：最新攻击时间（timestamp字段最大值）
			Value         float64 `json:"value"`           // 时间戳（毫秒级）
			ValueAsString string  `json:"value_as_string"` // 格式化后的最新攻击时间字符串
		} `json:"maxDate"`
	} `json:"buckets"` // 攻击源IP聚合桶列表（按攻击频次降序排列）
}

// SrcIpAggView 攻击源IP Top5聚合统计接口
func (IndexApi) SrcIpAggView(c *gin.Context) {
	// 构建ES聚合查询：主聚合按攻击源IP（srcIp）分组，取Top5；子聚合获取每个IP的最新攻击时间
	agg := elastic.NewTermsAggregation().
		Field("srcIp"). // 主聚合维度：攻击源IP（srcIp字段）
		Size(5). // 限制返回前5条高频攻击源IP
		SubAggregation("maxDate", elastic.NewMaxAggregation().Field("timestamp")) // 子聚合：最新攻击时间

	// 执行ES聚合查询：仅返回聚合结果
	res, err := global.ES.Search(es_models.AlertModel{}.Index()).
		Aggregation("agg", agg). // 绑定聚合查询配置
		Size(0). // 聚合查询无需返回文档
		Do(context.Background())
	if err != nil {
		logrus.Errorf("告警攻击源IP聚合查询失败 %s", err)
		response.FailWithMsg("告警查询失败", c)
		return
	}

	// 解析ES聚合结果到结构体
	var aggType SrcIpAggType
	err = json.Unmarshal(res.Aggregations["agg"], &aggType)
	if err != nil {
		logrus.Errorf("攻击源IP聚合结果json解析失败 %s %s", err, res.Aggregations["agg"])
		response.FailWithMsg("数据解析失败", c)
		return
	}

	// 构造标准化响应数据列表
	var list = make([]SrcIpAggResponse, 0, len(aggType.Buckets))
	for _, bucket := range aggType.Buckets {
		list = append(list, SrcIpAggResponse{
			SrcIp:         bucket.Key,                   // 攻击源IP
			Addr:          core.GetIpAddr(bucket.Key),   // 调用IP归属地解析工具获取归属地
			AttackCount:   bucket.DocCount,              // 该IP的总攻击次数
			NewAttackDate: bucket.MaxDate.ValueAsString, // 该IP的最新攻击时间
		})
	}

	// 返回Top5攻击源IP聚合统计结果
	response.OkWithData(list, c)
}
