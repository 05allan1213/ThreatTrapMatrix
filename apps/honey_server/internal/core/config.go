package core

// File: honey_server/core/config.go
// Description: 核心模块，提供配置文件读取功能

import (
	"honey_server/internal/config"
	"honey_server/internal/flags"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// ReadConfig 读取并解析配置文件，返回配置结构体指针
func ReadConfig() *config.Config {
	// 读取配置文件内容
	byteData, err := os.ReadFile(flags.Options.File)
	if err != nil {
		logrus.Fatalf("配置文件读取错误 %s", err)
		return nil
	}

	// 初始化配置结构体
	c := new(config.Config)

	// 将YAML数据解析到配置结构体
	err = yaml.Unmarshal(byteData, &c)
	if err != nil {
		logrus.Fatalf("配置文件配置错误 %s", err)
		return nil
	}

	return c
}

// SetConfig 设置并保存配置文件
func SetConfig(c *config.Config) error {
	byteData, _ := yaml.Marshal(c)
	err := os.WriteFile(flags.Options.File, byteData, 0644)
	return err
}
