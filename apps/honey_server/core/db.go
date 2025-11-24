package core

// File: honey_server/core/db.go
// Description: 数据库核心模块，实现MySQL数据库连接初始化、连接池配置及连接有效性检测功能

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB MySQL数据库配置结构体
type DB struct {
	DbName   string `yaml:"db_name"`  // 数据库名称
	Host     string `yaml:"host"`     // 数据库主机地址
	Port     int    `yaml:"port"`     // 数据库端口号
	User     string `yaml:"user"`     // 数据库用户名
	Password string `yaml:"password"` // 数据库密码
}

// InitDB 初始化MySQL数据库连接并配置连接池
func InitDB() (database *gorm.DB) {
	// 初始化数据库配置参数
	var db = DB{
		DbName:   "honey_db",
		Host:     "82.157.155.26",
		Port:     3306,
		User:     "ayp",
		Password: "801026qwe",
	}

	// 拼接MySQL的DSN（Data Source Name）连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		db.User,
		db.Password,
		db.Host,
		db.Port,
		db.DbName,
	)

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

	return
}
