package config

// File: honey_server/config/enter.go
// Description: 配置模块，定义应用配置结构体及资源配置相关方法

import "fmt"

// Config 应用整体配置结构体
type Config struct {
	DB DB `yaml:"db"` // 数据库配置信息
}

// DB 数据库连接配置结构体
type DB struct {
	DbName   string `yaml:"db_name"`  // 数据库名称
	Host     string `yaml:"host"`     // 数据库主机地址
	Port     int    `yaml:"port"`     // 数据库端口
	User     string `yaml:"user"`     // 数据库用户名
	Password string `yaml:"password"` // 数据库密码
}

// Dsn 生成数据库连接DSN字符串
func (cfg DB) Dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DbName,
	)
}
