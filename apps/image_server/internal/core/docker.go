package core

// File: image_server/core/docker.go
// Description: Docker客户端初始化工具，提供Docker SDK客户端的创建及初始化功能

import (
	"github.com/moby/moby/client"
	"github.com/sirupsen/logrus"
)

// InitDocker 初始化Docker客户端连接
func InitDocker() *client.Client {
	// 从环境变量配置创建Docker客户端，并自动协商API版本
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		// 客户端创建失败时记录致命错误并终止程序
		logrus.Fatalf("创建Docker客户端失败: %v", err)
	}
	return cli
}
