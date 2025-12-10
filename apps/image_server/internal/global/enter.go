package global

// File: image_server/global/enter.go
// Description: 全局变量模块，定义应用程序级别的全局共享变量

import (
	"image_server/internal/config"

	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 全局变量声明区
var (
	DB           *gorm.DB       // 全局数据库连接实例
	Redis        *redis.Client  // 全局Redis连接实例
	Config       *config.Config // 全局配置实例
	Log          *logrus.Entry  // 全局日志实例
	DockerClient *client.Client // 全局Docker客户端实例
)

var (
	Version   = "v1.0.3"
	Commit    = "d1560c9"
	BuildTime = "2025-12-10 11:07:04"
)
