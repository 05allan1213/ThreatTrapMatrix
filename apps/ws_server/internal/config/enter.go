package config

// File: ws_server/config/enter.go
// Description: 配置模块，定义应用配置结构体及资源配置相关方法

import "fmt"

// Config 应用整体配置结构体
type Config struct {
	Logger Logger `yaml:"logger"` // 日志配置信息
	System System `yaml:"system"` // 系统配置信息
	Jwt    Jwt    `yaml:"jwt"`    // jwt配置信息
	MQ     MQ     `yaml:"mq"`     // rabbitMQ配置信息
}

// Logger 日志配置结构体
type Logger struct {
	Format  string `yaml:"format"`  // 日志格式 [json|text]
	Level   string `yaml:"level"`   // 日志级别
	AppName string `yaml:"appName"` // 应用名称
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
	WsTopic           string `yaml:"wsTopic"`           // WebSocket服务Topic
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
