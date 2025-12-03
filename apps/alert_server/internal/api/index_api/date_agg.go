package index_api

// File: alert_server/api/index_api/date_agg.go
// Description: 首页时间维度聚合统计API接口

import (
	"alert_server/internal/es_models"
	"alert_server/internal/global"
	"alert_server/internal/utils/response"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

// DateAggResponse 时间维度（小时）聚合统计响应结构体，返回单个小时的告警统计信息
type DateAggResponse struct {
	Date  string `json:"date"`  // 小时时间标识（格式：yyyy-MM-dd HH）
	Count int    `json:"count"` // 该小时内的告警总数量
}

// DateAggView 按小时统计当天告警数量接口
func (IndexApi) DateAggView(c *gin.Context) {
	// 1. 确定聚合时间范围：默认当前自然日
	today := time.Now().Format("2006-01-02")
	startTime := today + " 00:00:00" // 当天开始时间（零点）
	endTime := today + " 23:59:59"   // 当天结束时间（23点59分59秒）

	// 2. 构建时间范围过滤条件：仅聚合当天的告警数据，排除历史日期干扰
	rangeQuery := elastic.NewRangeQuery("timestamp").
		Gte(startTime). // 大于等于当天开始时间
		Lte(endTime) // 小于等于当天结束时间

	// 3. 构建按小时聚合的日期直方图：确保返回完整24小时数据，无数据小时填充0
	hourlyAgg := elastic.NewDateHistogramAggregation().
		Field("timestamp"). // 聚合字段：告警时间戳（timestamp）
		Interval("hour"). // 聚合间隔：每小时聚合一次
		Format("yyyy-MM-dd HH"). // 聚合结果时间格式：精确到小时
		MinDocCount(0). // 最小文档数为0：无告警的小时也返回，计数填0
		ExtendedBounds(today+" 00", today+" 23") // 强制扩展边界：固定返回00-23小时，避免缺失时段

	// 4. 执行ES聚合查询：仅返回聚合结果
	searchResult, err := global.ES.Search(es_models.AlertModel{}.Index()).
		Query(elastic.NewBoolQuery().Filter(rangeQuery)). // 应用当天时间范围过滤
		Aggregation("hourly_counts", hourlyAgg). // 绑定小时聚合配置
		Size(0). // 聚合查询无需返回原始文档
		Do(context.Background())

	if err != nil {
		logrus.Errorf("ES小时维度聚合查询失败: %v", err)
		response.FailWithMsg("聚合查询失败", c)
		return
	}

	// 5. 解析ES聚合结果：获取小时直方图聚合数据
	agg, found := searchResult.Aggregations.DateHistogram("hourly_counts")
	if !found {
		logrus.Warn("未找到小时维度聚合结果，返回空24小时列表")
		response.OkWithData([]DateAggResponse{}, c)
		return
	}

	// 6. 转换聚合结果为标准化响应格式
	var list []DateAggResponse
	for _, bucket := range agg.Buckets {
		list = append(list, DateAggResponse{
			Date:  *bucket.KeyAsString,  // 小时时间（如：2024-10-01 14）
			Count: int(bucket.DocCount), // 该小时告警数量（无数据时为0）
		})
	}

	// 返回当天24小时告警时序统计结果
	response.OkWithData(list, c)
}
