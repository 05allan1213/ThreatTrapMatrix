package alert_api

// File: alert_server/api/alert_api/src_ip_agg.go
// Description: 源ip告警聚合查询API接口

import (
	"alert_server/internal/core"
	"alert_server/internal/es_models"
	"alert_server/internal/global"
	"alert_server/internal/middleware"
	"alert_server/internal/models"
	"alert_server/internal/utils/response"
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

// SrcIpAggRequest 源IP告警聚合查询请求参数结构体，包含分页参数和源IP筛选条件
type SrcIpAggRequest struct {
	models.PageInfo        // 嵌入分页基础参数
	SrcIp           string `form:"srcIp"` // 源IP
}

// SrcIpAggResponse 源IP告警聚合查询响应结构体，返回单源IP的聚合统计信息
type SrcIpAggResponse struct {
	SrcIp         string   `json:"srcIp"`         // 攻击源IP地址
	Addr          string   `json:"addr"`          // 源IP归属地
	SignatureList []string `json:"signatureList"` // 该源IP涉及的攻击类型列表
	AttackCount   int      `json:"attackCount"`   // 该源IP的总攻击次数
	HoneyIpCount  int      `json:"honeyIpCount"`  // 该源IP攻击的蜜罐IP数量
	NewAttackDate string   `json:"newAttackDate"` // 该源IP的最新攻击时间
	IsWhite       bool     `json:"isWhite"`       // 该源IP是否在白名单中（true：在白名单，false：不在）
}

// AggType ES源IP聚合结果解析结构体，对应ES聚合查询返回的JSON格式（包含子聚合结果）
type AggType struct {
	DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"` // 聚合结果误差上限
	SumOtherDocCount        int `json:"sum_other_doc_count"`         // 未返回的聚合桶数量
	Buckets                 []struct {
		Key       string `json:"key"`       // 聚合维度值（即源IP地址）
		DocCount  int    `json:"doc_count"` // 该源IP的攻击总次数（聚合计数）
		Signature struct { // 子聚合：攻击类型（signature.keyword）去重统计
			DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
			SumOtherDocCount        int `json:"sum_other_doc_count"`
			Buckets                 []struct {
				Key      string `json:"key"`       // 攻击类型描述（signature字段值）
				DocCount int    `json:"doc_count"` // 该攻击类型的出现次数
			} `json:"buckets"`
		} `json:"signature"`
		IpCount struct { // 子聚合：目标诱捕IP（destIp）去重计数（基数统计）
			Value int `json:"value"` // 去重后的诱捕IP数量
		} `json:"ipCount"`
		MaxDate struct { // 子聚合：最新攻击时间（timestamp字段最大值）
			Value         float64 `json:"value"`           // 时间戳（毫秒级）
			ValueAsString string  `json:"value_as_string"` // 格式化后的时间字符串
		} `json:"maxDate"`
	} `json:"buckets"` // 源IP聚合桶列表
}

// AllAggType ES全量源IP聚合结果解析结构体，仅用于统计聚合桶总数（分页总条数）
type AllAggType struct {
	DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int `json:"sum_other_doc_count"`
	Buckets                 []struct {
		Key      string `json:"key"`       // 源IP地址
		DocCount int    `json:"doc_count"` // 攻击次数
	} `json:"buckets"` // 全量源IP聚合桶列表（用于计算总条数）
}

// SrcIpAggView 源IP维度告警聚合查询接口
func (AlertApi) SrcIpAggView(c *gin.Context) {
	// 绑定并校验请求参数（分页参数 + 源IP筛选条件）
	cr := middleware.GetBind[SrcIpAggRequest](c)

	// 分页参数校验与默认值设置：避免非法参数导致聚合异常，控制单页返回量
	if cr.Limit <= 0 {
		cr.Limit = 10 // 默认每页10条聚合结果
	}
	if cr.Limit > 20 {
		cr.Limit = 10 // 限制最大单页条数为10，降低ES聚合计算压力
	}
	if cr.Page <= 0 {
		cr.Page = 1 // 默认查询第1页
	}

	// 计算聚合桶分页偏移量（用于桶排序分页，与文档分页逻辑一致）
	offset := (cr.Page - 1) * cr.Limit

	// 构建ES布尔查询：支持源IP精确筛选（为空时查询所有）
	query := elastic.NewBoolQuery()
	if cr.SrcIp != "" {
		query = query.Filter(elastic.NewTermQuery("srcIp", cr.SrcIp)) // 源IP筛选条件
	}

	// 构建ES聚合查询：按源IP分组，包含多个子聚合统计
	agg := elastic.NewTermsAggregation().Field("srcIp") // 主聚合：按srcIp字段分组
	// 子聚合1：按攻击类型（signature.keyword）分组，获取该源IP涉及的所有攻击类型
	agg.SubAggregation("signature", elastic.NewTermsAggregation().Field("signature.keyword"))
	// 子聚合2：统计该源IP攻击的诱捕IP数量（destIp字段基数统计，去重）
	agg.SubAggregation("ipCount", elastic.NewCardinalityAggregation().Field("destIp"))
	// 子聚合3：获取该源IP的最新攻击时间（timestamp字段最大值）
	agg.SubAggregation("maxDate", elastic.NewMaxAggregation().Field("timestamp"))
	// 子聚合4：桶排序分页：按最新攻击时间降序，配合offset和limit实现分页
	agg.SubAggregation("page", elastic.NewBucketSortAggregation().
		Sort("maxDate", false). // 按最新攻击时间降序（false=desc）
		From(offset). // 分页偏移量
		Size(cr.Limit)) // 单页聚合桶数量

	// 执行ES查询：聚合查询+全量聚合统计（用于获取总条数）
	res, err := global.ES.Search(es_models.AlertModel{}.Index()).
		Query(query). // 应用筛选条件
		Aggregation("agg", agg). // 主聚合（带分页的源IP聚合）
		Aggregation("allAgg", elastic.NewTermsAggregation().Field("srcIp")). // 全量聚合（用于统计总条数）
		Size(0). // 聚合查询无需返回文档，Size设为0提升性能
		Do(context.Background())
	if err != nil {
		logrus.Errorf("告警聚合查询失败 %s", err)
		response.FailWithMsg("告警聚合查询失败", c)
		return
	}

	// 解析全量聚合结果，获取聚合桶总数（即源IP总数量，用于分页总条数）
	var allAggType AllAggType
	err = json.Unmarshal(res.Aggregations["allAgg"], &allAggType)
	if err != nil {
		logrus.Errorf("全量聚合结果json解析失败 %s %s", err, res.Aggregations["allAgg"])
		response.FailWithMsg("数据解析失败", c)
		return
	}
	count := len(allAggType.Buckets) // 总聚合桶数（源IP总数）

	// 解析主聚合结果（带分页的源IP聚合数据）
	var aggType AggType
	err = json.Unmarshal(res.Aggregations["agg"], &aggType)
	if err != nil {
		logrus.Errorf("主聚合结果json解析失败 %s %s", err, res.Aggregations["agg"])
		response.FailWithMsg("数据解析失败", c)
		return
	}

	// 查询白名单IP列表，用于标记源IP是否在白名单
	var srcIPList []string
	for _, bucket := range aggType.Buckets {
		srcIPList = append(srcIPList, bucket.Key) // 收集当前页所有源IP
	}
	var whiteIpList []models.WhiteIPModel
	global.DB.Find(&whiteIpList, "ip in ?", srcIPList) // 批量查询白名单
	whiteIpMap := make(map[string]bool, len(whiteIpList))
	for _, model := range whiteIpList {
		whiteIpMap[model.IP] = true // 构建白名单IP映射，便于快速查询
	}

	// 构造聚合响应结果列表
	var list = make([]SrcIpAggResponse, 0, len(aggType.Buckets))
	for _, bucket := range aggType.Buckets {
		// 提取该源IP涉及的所有攻击类型（去重）
		var signatureList []string
		for _, sBucket := range bucket.Signature.Buckets {
			signatureList = append(signatureList, sBucket.Key)
		}

		// 组装响应数据
		list = append(list, SrcIpAggResponse{
			SrcIp:         bucket.Key,
			Addr:          core.GetIpAddr(bucket.Key), // 解析IP归属地
			SignatureList: signatureList,
			AttackCount:   bucket.DocCount,
			HoneyIpCount:  bucket.IpCount.Value,
			NewAttackDate: bucket.MaxDate.ValueAsString,
			IsWhite:       whiteIpMap[bucket.Key], // 从白名单映射判断状态
		})
	}

	// 返回标准化分页聚合响应
	response.OkWithList(list, int64(count), c)
}
