package core

// File: honey_node/core/config.go
// Description: 核心模块，提供配置文件读取功能

import (
	"honey_node/internal/config"
	"honey_node/internal/flags"
	"honey_node/internal/global"
	"os"

	"github.com/google/uuid"
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
	SetDefault(c)
	return c
}

// SetDefault 设置默认配置项
func SetDefault(c *config.Config) {
	if c.System.Uid == "" {
		c.System.Uid = uuid.New().String()
		SetConfig()
	}
}

// SetConfig 更新配置文件内容，并保存到文件中
func SetConfig() {
	// 将全局配置序列化为YAML格式
	byteData, err := yaml.Marshal(global.Config)
	if err != nil {
		logrus.Errorf("配置序列化失败 %s", err)
		return
	}

	// 将序列化后的配置写入文件
	err = os.WriteFile(flags.Options.File, byteData, 0666)
	if err != nil {
		logrus.Errorf("配置文件写入错误 %s", err)
		return
	}

	// 记录配置文件更新成功的日志
	logrus.Infof("%s 配置文件更新成功", flags.Options.File)
}
