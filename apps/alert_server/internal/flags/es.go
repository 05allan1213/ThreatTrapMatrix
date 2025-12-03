package flags

// File: alert_server/flags/es.go
// Description: ES索引管理模块，负责告警索引的存在性检查、旧索引删除及新索引创建（含映射配置）

import (
	"alert_server/internal/es_models"
	"alert_server/internal/global"
	"context"

	"github.com/sirupsen/logrus"
)

// EsIndex 初始化ES告警索引：检查索引是否存在，存在则删除旧索引，创建新索引并应用映射配置
func EsIndex() {
	// 获取告警索引名（从告警数据模型中读取配置的索引名称）
	index := es_models.AlertModel{}.Index()

	// 检查目标索引是否已存在
	ok, err := global.ES.IndexExists(index).Do(context.Background())
	if err != nil {
		logrus.Errorf("获取索引错误 %s", err)
		return
	}

	// 索引已存在时，先删除旧索引（避免映射变更导致的兼容性问题）
	if ok {
		logrus.Infof("存在索引 删除索引 %s", index)
		_, err = global.ES.DeleteIndex(index).Do(context.Background())
		if err != nil { // 补充原有逻辑未显式处理的删除错误（不阻断，仅日志）
			logrus.Errorf("删除旧索引 %s 失败: %s", index, err)
		}
	}

	// 创建新索引，并应用告警模型中定义的映射配置
	logrus.Infof("创建索引 %s", index)
	response, err := global.ES.CreateIndex(index).
		Body(es_models.AlertModel{}.Mappings()). // 加载嵌入的索引映射配置
		Do(context.Background())
	if err != nil {
		logrus.Errorf("创建索引错误 %s", err)
		return
	}
	logrus.Infof("创建索引成功 %v", response)
}
