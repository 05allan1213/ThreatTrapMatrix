package core

// File: image_server/core/config.go
// Description: 核心模块，提供配置文件读取功能

import (
	"image_server/internal/config"
	"image_server/internal/flags"
	"image_server/internal/global"
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

// SetConfig 将配置结构体写入配置文件
func SetConfig() {
	byteData, err := yaml.Marshal(global.Config)
	if err != nil {
		logrus.Errorf("配置序列化失败 %s", err)
		return
	}
	err = os.WriteFile(flags.Options.File, byteData, 0666)
	if err != nil {
		logrus.Errorf("配置文件写入错误 %s", err)
		return
	}
	logrus.Infof("%s 配置文件更新成功", flags.Options.File)
}
