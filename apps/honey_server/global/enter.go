package global

// File: honey_server/global/enter.go
// Description: 全局变量模块，定义应用程序级别的全局共享变量

import (
	"ThreatTrapMatrix/apps/honey_server/config"

	"gorm.io/gorm"
)

// 全局变量声明区
var (
	DB     *gorm.DB       // 全局数据库连接实例
	Config *config.Config // 全局配置实例
)
