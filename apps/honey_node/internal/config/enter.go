package config

// File: honey_node/config/enter.go
// Description: 配置模块，定义应用配置结构体及资源配置相关方法

// Config 应用整体配置结构体
type Config struct {
	Logger Logger `yaml:"logger"` // 日志配置信息
	System System `yaml:"system"` // 系统配置信息
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
}
