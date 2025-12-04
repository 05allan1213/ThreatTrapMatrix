package config

import (
	"fmt"
)

// File: honey_node/config/enter.go
// Description: 配置模块，定义应用配置结构体及资源配置相关方法

// Config 应用整体配置结构体
type Config struct {
	Logger            Logger   `yaml:"logger"`            // 日志配置信息
	System            System   `yaml:"system"`            // 系统配置信息
	FilterNetworkList []string `yaml:"filterNetworkList"` // 网卡过滤列表
	MQ                MQ       `yaml:"mq"`                // rabbitMQ配置信息
	DB                DB       `yaml:"db"`                // 数据库配置信息
}

// Logger 日志配置结构体
type Logger struct {
	Format  string `yaml:"format"`  // 日志格式 [json|text]
	Level   string `yaml:"level"`   // 日志级别
	AppName string `yaml:"appName"` // 应用名称
}

// System 系统配置结构体
type System struct {
	GrpcManageAddr string `yaml:"grpcManageAddr"` // gRPC管理服务监听地址
	Network        string `yaml:"network"`        // 网卡
	Uid            string `yaml:"uid"`            // 节点uid
	EvePath        string `yaml:"evePath"`        // eve文件路径
}

// rabbitMQ 配置结构体
type MQ struct {
	User                    string `yaml:"user"`                    // 用户名
	Password                string `yaml:"password"`                // 密码
	Host                    string `yaml:"host"`                    // 主机地址
	Port                    int    `yaml:"port"`                    // 端口号
	CreateIpExchangeName    string `yaml:"createIpExchangeName"`    // 创建IP交换机名称
	DeleteIpExchangeName    string `yaml:"deleteIpExchangeName"`    // 删除IP交换机名称
	BindPortExchangeName    string `yaml:"bindPortExchangeName"`    // 绑定端口交换机名称
	BatchDeployExchangeName string `yaml:"batchDeployExchangeName"` // 批量部署交换机名称
	Ssl                     bool   `yaml:"ssl"`                     // 是否使用SSL
	ClientCertificate       string `yaml:"clientCertificate"`       // 客户端证书
	ClientKey               string `yaml:"clientKey"`               // 客户端密钥
	CaCertificate           string `yaml:"caCertificate"`           // CA证书
	AlertTopic              string `yaml:"alertTopic"`              // 告警Topic名称
	BatchDeployStatusTopic  string `yaml:"batchDeployStatusTopic"`  // 批量部署上报状态的topic
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

// DB SQLite数据库配置结构体
type DB struct {
	DbName          string `yaml:"db_name"`         // 数据库名称
	MaxIdleConns    int    `yaml:"maxIdleConns"`    // 最大空闲连接数
	MaxOpenConns    int    `yaml:"maxOpenConns"`    // 最大连接数
	ConnMaxLifetime int    `yaml:"connMaxLifetime"` // 连接最大生命周期（秒）
}
