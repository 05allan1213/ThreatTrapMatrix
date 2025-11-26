package core

// File: honey_server/core/db.go
// Description: 数据库核心模块，实现MySQL数据库连接初始化、连接池配置及连接有效性检测功能

import (
	"ThreatTrapMatrix/apps/honey_server/internal/global"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitDB 初始化MySQL数据库连接并配置连接池
func InitDB() (db *gorm.DB) {
	cfg := global.Config.DB
	db, err := gorm.Open(mysql.Open(cfg.Dsn()), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // 迁移时禁用外键约束，提高灵活性
	})
	if err != nil {
		logrus.Fatalf("数据库连接失败 %s", err)
		return
	}

	// 获取底层sql.DB实例以配置连接池
	sqlDB, err := db.DB()
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
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 10
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 100
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 10000
	}
	logrus.Infof("最大空闲数 %d", cfg.MaxIdleConns)
	logrus.Infof("最大连接数 %d", cfg.MaxOpenConns)
	logrus.Infof("超时时间 %s", time.Duration(cfg.ConnMaxLifetime)*time.Second)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	logrus.Infof("数据库连接成功")
	return
}

var (
	db        *gorm.DB
	onceMysql sync.Once
)

// GetDB 获取数据库连接实例（单例模式）
func GetDB() *gorm.DB {
	onceMysql.Do(func() {
		db = InitDB()
	})
	return db
}
