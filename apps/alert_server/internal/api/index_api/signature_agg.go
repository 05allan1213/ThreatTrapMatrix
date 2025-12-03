package index_api

// File: alert_server/api/alert_api/signature_agg.go
// Description: 告警攻击类型聚合统计API接口层，为首页提供攻击类型排行展示

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

// SignatureAggResponse 攻击类型聚合统计响应结构体，返回单种攻击类型的统计信息
type SignatureAggResponse struct {
	Signature string `json:"signature"` // 攻击类型描述（对应告警规则的signature字段）
	Count     int    `json:"count"`     // 该攻击类型的总出现次数（攻击频次）
}

// AggType ES攻击类型聚合结果解析结构体，对应ES terms聚合返回的JSON格式
type AggType struct {
	DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"` // 聚合结果误差上限
	SumOtherDocCount        int `json:"sum_other_doc_count"`         // 未返回的聚合桶数量
	Buckets                 []struct {
		Key      string `json:"key"`       // 聚合维度值（即攻击类型描述，signature.keyword字段值）
		DocCount int    `json:"doc_count"` // 该攻击类型的总出现次数（聚合计数）
	} `json:"buckets"` // 攻击类型聚合桶列表（按出现频次降序排列）
}

// SignatureAggView 攻击类型Top5聚合统计接口
func (IndexApi) SignatureAggView(c *gin.Context) {
	// 构建ES聚合查询：按攻击类型（signature.keyword）分组，取Top5（size=5），按出现频次降序排列
	agg := elastic.NewTermsAggregation().
		Size(5). // 限制返回前5条高频攻击类型
		Field("signature.keyword") // 按signature.keyword字段聚合

	// 执行ES聚合查询：仅返回聚合结果，不返回具体文档
	res, err := global.ES.Search(es_models.AlertModel{}.Index()).
		Aggregation("agg", agg). // 绑定聚合查询配置
		Size(0). // 聚合查询无需返回文档
		Do(context.Background())
	if err != nil {
		logrus.Errorf("告警攻击类型聚合查询失败 %s", err)
		response.FailWithMsg("告警查询失败", c)
		return
	}

	// 解析ES聚合结果到结构体
	var data AggType
	err = json.Unmarshal(res.Aggregations["agg"], &data)
	if err != nil {
		logrus.Errorf("攻击类型聚合结果解析失败 %s", err)
		response.FailWithMsg("数据解析失败", c)
		return
	}

	// 构造标准化响应数据列表
	var list = make([]SignatureAggResponse, 0, len(data.Buckets))
	for _, bucket := range data.Buckets {
		list = append(list, SignatureAggResponse{
			Signature: bucket.Key,      // 攻击类型描述
			Count:     bucket.DocCount, // 攻击类型出现次数
		})
	}

	// 返回聚合统计结果（Top5攻击类型及频次）
	response.OkWithData(list, c)
}
