package alert_api

// File: alert_server/api/alert_api/signature_options.go
// Description: 攻击类型optionsAPI接口

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

// SignatureAggType ES聚合查询返回的签名聚合结果结构
type SignatureAggType struct {
	DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"` // 文档计数误差上限
	SumOtherDocCount        int `json:"sum_other_doc_count"`         // 未纳入聚合的文档总数
	Buckets                 []struct {
		Key      string `json:"key"`       // 聚合维度值（告警签名）
		DocCount int    `json:"doc_count"` // 该签名对应的文档数量
	} `json:"buckets"` // 聚合桶列表
}

// SignatureOptionsResponse 告警签名选项响应结构体
type SignatureOptionsResponse struct {
	Label string `json:"label"` // 展示文本
	Value string `json:"value"` // 选项值
}

// SignatureOptionsView 告警签名选项查询接口处理函数
func (AlertApi) SignatureOptionsView(c *gin.Context) {
	// 创建ES Terms聚合条件：按signature.keyword字段聚合，最多返回10000个唯一值
	// keyword类型确保按完整字符串精确聚合，避免分词导致的聚合错误
	agg := elastic.NewTermsAggregation().Size(10000).Field("signature.keyword")

	// 执行ES聚合查询：指定告警索引，仅返回聚合结果
	res, err := global.ES.Search(es_models.AlertModel{}.Index()).
		Aggregation("agg", agg). // 绑定聚合条件，命名为"agg"
		Size(0).Do(context.Background())
	if err != nil {
		logrus.Errorf("告警查询失败 %s", err)
		response.FailWithMsg("告警查询失败", c)
		return
	}

	// 解析ES聚合结果：将agg聚合结果反序列化为SignatureAggType结构体
	var data SignatureAggType
	err = json.Unmarshal(res.Aggregations["agg"], &data)
	if err != nil {
		response.FailWithMsg("解析失败", c)
		return
	}

	// 组装前端所需的签名选项列表
	var list = make([]SignatureOptionsResponse, 0)
	for _, bucket := range data.Buckets {
		list = append(list, SignatureOptionsResponse{
			Label: bucket.Key, // 展示文本使用签名值
			Value: bucket.Key, // 选项值使用签名值
		})
	}

	// 返回成功响应，携带签名选项列表
	response.OkWithData(list, c)
}
