package alert_api

// File: alert_server/api/alert_api/alert_list.go
// Description: 告警列表查询API接口

import (
	"alert_server/internal/es_models"
	"alert_server/internal/global"
	"alert_server/internal/middleware"
	"alert_server/internal/models"
	"alert_server/internal/utils/response"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

// ListRequest 告警列表查询请求参数结构体，包含分页参数和多维度筛选条件
type ListRequest struct {
	models.PageInfo        // 嵌入分页基础参数
	SrcIp           string `form:"srcIp"` // 攻击源IP
	DestIp          string `form:"destIp"` // 攻击目标IP
	DestPort        int    `form:"destPort"` // 攻击目标端口
	ServiceName     string `form:"serviceName"` // 关联服务名称
	Signature       string `form:"signature"` // 攻击类型
	Level           int8   `form:"level"` // 告警级别
	StartTime       string `form:"startTime"` // 告警开始时间
	EndTime         string `form:"endTime"` // 告警结束时间
}

// ListView 告警列表查询接口，支持多条件组合筛选、时间范围解析、分页控制，从ES查询并返回标准化响应
func (AlertApi) ListView(c *gin.Context) {
	// 绑定并校验查询参数
	cr := middleware.GetBind[ListRequest](c)

	// 分页参数校验与默认值设置：避免非法参数导致查询异常，控制单页查询量
	if cr.Limit <= 0 {
		cr.Limit = 10 // 默认每页10条数据
	}
	if cr.Limit > 20 {
		cr.Limit = 10 // 限制最大单页条数为10，降低ES查询压力
	}
	if cr.Page <= 0 {
		cr.Page = 1 // 默认查询第1页
	}

	// 计算ES查询偏移量
	offset := (cr.Page - 1) * cr.Limit

	// 构建ES布尔查询
	query := elastic.NewBoolQuery()

	// 1. 源IP筛选：精确匹配（仅当参数不为空时添加条件）
	if cr.SrcIp != "" {
		query = query.Filter(elastic.NewTermQuery("srcIp", cr.SrcIp))
	}

	// 2. 目标IP筛选：精确匹配（仅当参数不为空时添加条件）
	if cr.DestIp != "" {
		query = query.Filter(elastic.NewTermQuery("destIp", cr.DestIp))
	}

	// 3. 目标端口筛选：精确匹配（仅当参数不为0时添加条件，0表示不筛选）
	if cr.DestPort != 0 {
		query = query.Filter(elastic.NewTermQuery("destPort", cr.DestPort))
	}

	// 4. 服务名称筛选：基于keyword字段精确匹配（避免分词导致的模糊匹配问题）
	if cr.ServiceName != "" {
		query = query.Filter(elastic.NewTermQuery("serviceName.keyword", cr.ServiceName))
	}

	// 5. 攻击类型筛选：基于match查询实现分词模糊匹配（支持部分关键词查询）
	if cr.Signature != "" {
		query = query.Filter(elastic.NewMatchQuery("signature", cr.Signature))
	}

	// 6. 告警级别筛选：精确匹配（仅当参数不为0时添加条件，0表示不筛选）
	if cr.Level != 0 {
		query = query.Filter(elastic.NewTermQuery("level", cr.Level))
	}

	// 7. 时间范围筛选：支持开始时间/结束时间单独或组合筛选，自动兼容多时间格式
	if cr.StartTime != "" || cr.EndTime != "" {
		rangeQuery := elastic.NewRangeQuery("timestamp") // 基于告警时间字段筛选

		// 解析开始时间：支持yyyy-MM-dd HH:mm:ss和yyyy-MM-dd两种格式
		if cr.StartTime != "" {
			if startTime, err := parseTime(cr.StartTime); err == nil {
				rangeQuery = rangeQuery.Gte(startTime) // 大于等于开始时间
			} else {
				logrus.Warnf("无效的开始时间格式: %s, 错误: %v", cr.StartTime, err) // 日志警告，不阻断查询
			}
		}

		// 解析结束时间：支持yyyy-MM-dd HH:mm:ss和yyyy-MM-dd两种格式
		if cr.EndTime != "" {
			if endTime, err := parseTime(cr.EndTime); err == nil {
				rangeQuery = rangeQuery.Lte(endTime) // 小于等于结束时间
			} else {
				logrus.Warnf("无效的结束时间格式: %s, 错误: %v", cr.EndTime, err) // 日志警告，不阻断查询
			}
		}

		query = query.Filter(rangeQuery)
	}

	// 设置默认排序规则：按告警时间戳降序（最新告警在前）
	sortBy := "timestamp"

	// 执行ES查询：指定索引、查询条件、排序规则、分页参数
	res, err := global.ES.Search(es_models.AlertModel{}.Index()). // 从告警模型配置的ES索引查询
		Query(query). // 应用所有筛选条件
		Sort(sortBy, false). // 降序排序（false表示desc，true表示asc）
		Size(cr.Limit). // 单页数据条数
		From(offset). // 查询偏移量
		Do(context.Background()) // 带上下文执行查询

	if err != nil {
		logrus.Errorf("告警查询失败 %s", err)
		response.FailWithMsg("告警查询失败", c)
		return
	}

	// 解析查询结果：总条数 + 告警数据列表
	count := res.Hits.TotalHits.Value                    // 符合筛选条件的告警总条数
	var list = make([]es_models.AlertModel, 0, cr.Limit) // 初始化结果切片

	for _, hit := range res.Hits.Hits {
		var data es_models.AlertModel
		// 将ES文档源数据（JSON格式）解析为告警模型结构体
		err = json.Unmarshal(hit.Source, &data)
		if err != nil {
			logrus.Errorf("json解析失败 %s %s %s", err, hit.Source, hit.Id)
			continue // 解析失败跳过当前数据，继续处理下一条
		}
		data.ID = hit.Id // 补充ES文档唯一ID到结果中，便于后续详情查询等操作
		list = append(list, data)
	}

	// 返回标准化分页响应
	response.OkWithList(list, count, c)
}

// parseTime 解析时间字符串，支持两种常用格式（yyyy-MM-dd HH:mm:ss / yyyy-MM-dd），解析失败返回错误
func parseTime(timeStr string) (string, error) {
	// 尝试解析为完整时间格式（yyyy-MM-dd HH:mm:ss）
	if t, err := time.Parse(time.DateTime, timeStr); err == nil {
		return t.Format(time.DateTime), nil
	}

	// 尝试解析为日期格式（yyyy-MM-dd），自动补全时间部分为00:00:00
	if t, err := time.Parse(time.DateOnly, timeStr); err == nil {
		return t.Format(time.DateTime), nil
	}

	// 不支持的时间格式，返回错误
	return "", fmt.Errorf("不支持的时间格式: %s，仅支持 yyyy-MM-dd HH:mm:ss 或 yyyy-MM-dd", timeStr)
}
