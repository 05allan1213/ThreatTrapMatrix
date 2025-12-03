package config

// File: alert_server/config/enter.go
// Description: 配置模块，定义应用配置结构体及资源配置相关方法

import "fmt"

// Config 应用整体配置结构体
type Config struct {
	DB        DB       `yaml:"db"`        // 数据库配置信息
	Logger    Logger   `yaml:"logger"`    // 日志配置信息
	Redis     Redis    `yaml:"redis"`     // redis配置信息
	System    System   `yaml:"system"`    // 系统配置信息
	Jwt       Jwt      `yaml:"jwt"`       // jwt配置信息
	WhiteList []string `yaml:"whiteList"` // 路由白名单
	MQ        MQ       `yaml:"mq"`        // rabbitMQ配置信息
	ES        ES       `yaml:"es"`        // elasticSearch配置信息
	Alert     Alert    `yaml:"alert"`     // 告警配置信息
}

// DB 数据库连接配置结构体
type DB struct {
	DbName          string `yaml:"db_name"`         // 数据库名称
	Host            string `yaml:"host"`            // 数据库主机地址
	Port            int    `yaml:"port"`            // 数据库端口
	User            string `yaml:"user"`            // 数据库用户名
	Password        string `yaml:"password"`        // 数据库密码
	MaxIdleConns    int    `yaml:"maxIdleConns"`    // 数据库最大空闲连接数
	MaxOpenConns    int    `yaml:"maxOpenConns"`    // 数据库最大打开连接数
	ConnMaxLifetime int    `yaml:"connMaxLifetime"` // 数据库连接最大生命周期
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

// Logger 日志配置结构体
type Logger struct {
	Format  string `yaml:"format"`  // 日志格式 [json|text]
	Level   string `yaml:"level"`   // 日志级别
	AppName string `yaml:"appName"` // 应用名称
}

// Redis 配置结构体
type Redis struct {
	Addr     string // Redis地址
	Password string // Redis密码
	DB       int    // Redis数据库索引
}

// System 系统配置结构体
type System struct {
	WebAddr string `yaml:"webAddr"` // Web服务监听地址
	Mode    string `yaml:"mode"`    // 运行模式 [debug|release|test]
}

// Jwt 配置结构体
type Jwt struct {
	Expires int    `yaml:"expires"` // token过期时间,单位秒
	Issuer  string `yaml:"issuer"`  // token签发者
	Secret  string `yaml:"secret"`  // token密钥
}

// rabbitMQ 配置结构体
type MQ struct {
	User              string `yaml:"user"`              // 用户名
	Password          string `yaml:"password"`          // 密码
	Host              string `yaml:"host"`              // 主机地址
	Port              int    `yaml:"port"`              // 端口号
	Ssl               bool   `yaml:"ssl"`               // 是否使用SSL
	ClientCertificate string `yaml:"clientCertificate"` // 客户端证书
	ClientKey         string `yaml:"clientKey"`         // 客户端密钥
	CaCertificate     string `yaml:"caCertificate"`     // CA证书
}

// Addr 获取rabbitMQ地址
func (m MQ) Addr() string {
	// 判断是否使用SSL
	if m.Ssl {
		return fmt.Sprintf("amqps://%s:%s@%s:%d/", // 使用SSL
			m.User,
			m.Password,
			m.Host,
			m.Port,
		)
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%d/",
		m.User,
		m.Password,
		m.Host,
		m.Port,
	)
}

// ElasticSearch 配置结构体
type ES struct {
	Addr     string `yaml:"addr"`     // ElasticSearch地址
	Username string `yaml:"username"` // ElasticSearch用户名
	Password string `yaml:"password"` // ElasticSearch密码
}

// Alert 配置结构体
type Alert struct {
	AlertIndex string `yaml:"alertIndex"` // 告警索引名称
	AlertTopic string `yaml:"alertTopic"` // 告警Topic名称
}
