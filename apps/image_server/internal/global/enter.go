package global

// File: honey_server/global/enter.go
// Description: 全局变量模块，定义应用程序级别的全局共享变量

import (
	"ThreatTrapMatrix/apps/image_server/internal/config"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 全局变量声明区
var (
	DB     *gorm.DB       // 全局数据库连接实例
	Redis  *redis.Client  // 全局Redis连接实例
	Config *config.Config // 全局配置实例
	Log    *logrus.Entry  // 全局日志实例
)

var (
	Version   = "v1.0.1"
	Commit    = "a29bb955"
	BuildTime = "2025-11-24 19:45:58"
)
