package core

// File: alert_server/core/es.go
// Description: Elasticsearch核心连接模块，负责基于配置参数初始化ES客户端，建立认证连接并返回可用实例

import (
	"alert_server/internal/global"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

// ConnectEs 初始化Elasticsearch客户端连接，支持基础认证，返回可用的ES客户端实例
func ConnectEs() *elastic.Client {
	cfg := global.Config.ES // 读取ES连接配置（地址、用户名、密码）
	var err error

	// 关闭ES节点嗅探模式
	sniffOpt := elastic.SetSniff(false)

	// 构建ES客户端，配置连接地址、认证信息及嗅探参数
	c, err := elastic.NewClient(
		elastic.SetURL(cfg.Addr),                         // 指定ES服务地址
		sniffOpt,                                         // 应用嗅探配置
		elastic.SetBasicAuth(cfg.Username, cfg.Password), // 配置基础认证用户名密码
	)
	if err != nil {
		logrus.Fatalf("Elasticsearch连接失败 %s", err.Error())
	}
	logrus.Infof("Elasticsearch连接成功")
	return c
}
