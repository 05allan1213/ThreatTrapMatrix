package core

// File: honey_server/core/db.go
// Description: 数据库核心模块，实现MySQL数据库连接初始化、连接池配置及连接有效性检测功能

import (
	"sync"
	"time"

	"ThreatTrapMatrix/apps/honey_server/global"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitDB 初始化MySQL数据库连接并配置连接池
func InitDB() (database *gorm.DB) {
	dsn := global.Config.DB.Dsn()
	// 创建MySQL驱动的GORM连接实例
	dialector := mysql.Open(dsn)
	database, err := gorm.Open(dialector, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // 迁移时禁用外键约束，提高灵活性
	})
	if err != nil {
		logrus.Fatalf("数据库连接失败 %s", err)
		return
	}

	// 获取底层sql.DB实例以配置连接池
	sqlDB, err := database.DB()
	if err != nil {
		logrus.Fatalf("获取数据库连接实例失败 %s", err)
		return
	}

	// 检测数据库连接有效性
	err = sqlDB.Ping()
	if err != nil {
		logrus.Fatalf("数据库连接有效性检测失败 %s", err)
		return
	}

	// 配置数据库连接池参数
	sqlDB.SetMaxIdleConns(10)           // 设置连接池最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 设置连接池最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 设置连接的最大生命周期

	logrus.Infof("数据库连接成功")
	return
}

var (
	db   *gorm.DB
	once sync.Once
)

// GetDB 获取数据库连接实例（单例模式）
func GetDB() *gorm.DB {
	once.Do(func() {
		db = InitDB()
	})
	return db
}
