package alert_api

// File: alert_server/api/alert_api/remove.go
// Description: 告警记录删除API接口

import (
	"alert_server/internal/es_models"
	"alert_server/internal/global"
	"alert_server/internal/middleware"
	"alert_server/internal/utils"
	"alert_server/internal/utils/response"
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

// RemoveRequest 告警记录删除请求参数结构体，支持两种删除维度（互斥或同时生效）
type RemoveRequest struct {
	ID    string `json:"id"`    // 单个告警记录ID
	SrcIp string `json:"srcIp"` // 源IP（删除该IP对应的所有告警记录）
}

// RemoveView 告警记录删除接口，支持单个ID删除或按源IP批量删除，自动整合待删除ID列表后执行批量删除
func (AlertApi) RemoveView(c *gin.Context) {
	log := middleware.GetLog(c)

	// 绑定并校验删除请求参数
	cr := middleware.GetBind[RemoveRequest](c)

	log.WithFields(map[string]interface{}{
		"alert_id": cr.ID,
		"src_ip":   cr.SrcIp,
	}).Info("alert removal request received") // 收到告警记录删除请求

	var idList []string // 待删除的告警记录ID列表（整合单个ID和源IP对应的所有ID）

	// 1. 处理单个告警ID删除：参数不为空时添加到ID列表
	if cr.ID != "" {
		idList = append(idList, cr.ID)
		log.WithFields(map[string]interface{}{
			"alert_id": cr.ID,
		}).Debug("added single alert ID to removal list") // 添加单个ID到删除列表
	}

	// 2. 处理按源IP批量删除：参数不为空时，查询该IP下所有告警记录的ID
	if cr.SrcIp != "" {
		log.WithFields(map[string]interface{}{
			"src_ip": cr.SrcIp,
		}).Debug("searching alerts by source IP for removal") // 查询该IP下的所有告警记录的ID

		// 从ES查询该源IP对应的所有告警记录（单次最多查询10000条，避免查询量过大）
		res, err := global.ES.Search(es_models.AlertModel{}.Index()).
			Query(elastic.NewTermQuery("srcIp", cr.SrcIp)). // 按源IP精确筛选
			Size(10000). // 限制单次查询最大条数
			Do(context.Background())
		if err != nil {
			log.WithFields(map[string]interface{}{
				"src_ip": cr.SrcIp,
				"error":  err,
			}).Error("failed to search alerts by source IP") // 按源IP搜索告警失败
			response.FailWithMsg("告警查询失败", c)
			return
		}

		// 提取查询结果中的所有告警ID，添加到待删除列表
		for _, hit := range res.Hits.Hits {
			idList = append(idList, hit.Id)
		}
		log.WithFields(map[string]interface{}{
			"src_ip":      cr.SrcIp,
			"found_count": len(res.Hits.Hits),
		}).Debug("found alerts for source IP removal") // 按源IP搜索到告警
	}

	// 去重ID列表（防止重复删除）
	idList = utils.Unique(idList)

	// 校验待删除ID列表是否为空（无有效删除目标时返回失败）
	if len(idList) == 0 {
		log.Warn("no alert records found for removal") // 无有效删除目标
		response.FailWithMsg("不存在的告警记录", c)
		return
	}

	// 执行批量删除操作
	log.WithFields(map[string]interface{}{
		"total_ids":  len(idList),
		"sample_ids": idList,
	}).Debug("preparing to batch remove alerts") // 准备批量删除告警记录

	if err := BatchRemove(idList); err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
			"ids":   idList,
		}).Error("failed to remove alert records") // 删除告警记录失败
		response.FailWithMsg("告警记录删除失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"removed_count": len(idList),
	}).Info("alert records removed successfully") // 告警记录删除成功
	response.OkWithMsg("告警记录删除成功", c)
}

// BatchRemove 批量删除ES中的告警记录，基于ES Bulk API实现高效批量操作，支持空列表安全处理
func BatchRemove(ids []string) error {
	// 空列表直接返回，避免无效操作
	if len(ids) == 0 {
		return nil
	}

	// 初始化ES批量操作请求
	bulkRequest := global.ES.Bulk()
	indexName := es_models.AlertModel{}.Index() // 获取告警数据存储的ES索引名

	// 为每个有效ID添加删除请求（过滤空ID，避免无效操作）
	for _, id := range ids {
		if id != "" {
			// 构建单个删除请求：指定索引和文档ID
			req := elastic.NewBulkDeleteRequest().
				Index(indexName).
				Id(id)
			bulkRequest = bulkRequest.Add(req) // 添加到批量请求中
		}
	}

	// 执行批量删除：Refresh("true")表示删除后实时刷新索引，确保删除立即生效
	res, err := bulkRequest.Refresh("true").Do(context.Background())
	if err != nil {
		logrus.Errorf("ES 批量删除失败: %v", err)
		return err
	}

	// 检查批量操作是否存在部分失败（部分ID删除失败时返回错误）
	if res.Errors {
		var failedIDs []string
		for _, item := range res.Failed() {
			failedIDs = append(failedIDs, item.Id)
			logrus.Errorf("ID %s 删除失败: %v", item.Id, item.Error)
		}
		return fmt.Errorf("部分删除失败，失败 ID: %v", failedIDs)
	}

	logrus.Infof("批量删除成功，共删除 %d 条数据", len(res.Succeeded()))
	return nil
}
