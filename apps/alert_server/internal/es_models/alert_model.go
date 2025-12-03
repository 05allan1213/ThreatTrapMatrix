package es_models

// File: alert_server/models/es_models/alert_model.go
// Description: Elasticsearch告警数据模型模块，定义告警数据存储结构、ES索引名及索引映射配置

import (
	"alert_server/internal/global"
	_ "embed"
)

// AlertModel Elasticsearch告警数据存储结构体
type AlertModel struct {
	ID          string `json:"id"`          // 告警唯一标识
	NodeUid     string `json:"nodeUid"`     // 节点唯一标识
	SrcIp       string `json:"srcIp"`       // 攻击源IP地址
	SrcPort     int    `json:"srcPort"`     // 攻击源端口
	DestIp      string `json:"destIp"`      // 攻击目标IP地址
	DestPort    int    `json:"destPort"`    // 攻击目标端口
	Timestamp   string `json:"timestamp"`   // 告警发生时间
	Signature   string `json:"signature"`   // 告警规则描述
	Level       int8   `json:"level"`       // 告警级别
	Body        string `json:"body"`        // HTTP响应体内容（仅HTTP相关告警有效）
	Payload     string `json:"payload"`     // 告警关联数据包请求载荷
	ServiceID   uint   `json:"serviceID"`   // 服务ID
	ServiceName string `json:"serviceName"` // 服务名称
}

// Index 获取告警数据在Elasticsearch中的存储索引名，从全局配置读取
func (alert AlertModel) Index() string {
	return global.Config.Alert.AlertIndex
}

//go:embed alert_mapping.json
var alertMapping string // 嵌入ES索引映射配置文件，避免硬编码

// Mappings 返回Elasticsearch告警索引的映射配置，用于索引初始化
func (alert AlertModel) Mappings() string {
	return alertMapping
}
