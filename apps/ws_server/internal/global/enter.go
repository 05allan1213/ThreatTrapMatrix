package global

// File: ws_server/global/enter.go
// Description: 全局变量模块，定义应用程序级别的全局共享变量

import (
	"ws_server/internal/config"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// 全局变量声明区
var (
	Config *config.Config // 全局配置实例
	Log    *logrus.Entry  // 全局日志实例
	Queue  *amqp.Channel  // 全局队列实例
)

var (
	Version   = "v1.0.1"
	Commit    = "a29bb955"
	BuildTime = "2025-11-24 19:45:58"
)
