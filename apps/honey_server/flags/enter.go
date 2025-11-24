package flags

// File: honey_server/flags/enter.go
// Description: 命令行参数解析模块，负责定义、解析命令行参数并执行对应操作

import (
	"flag"
	"os"

	"ThreatTrapMatrix/apps/honey_server/global"

	"github.com/sirupsen/logrus"
)

// FlagOptions 命令行参数选项结构体，存储解析后的命令行参数值
type FlagOptions struct {
	File    string // 配置文件路径
	Version bool   // 版本信息
	DB      bool   // 数据库表结构迁移
}

// Options 全局命令行参数实例
var Options FlagOptions

// init 初始化函数，用于定义并解析命令行参数
func init() {
	// 定义-f参数：指定配置文件路径
	flag.StringVar(&Options.File, "f", "settings.yaml", "配置文件路径")
	// 定义-v参数：控制是否打印版本信息
	flag.BoolVar(&Options.Version, "v", false, "打印当前版本")
	// 定义-db参数：控制是否执行数据库表结构迁移
	flag.BoolVar(&Options.DB, "db", false, "迁移表结构")
	// 解析命令行参数，填充到Options结构体中
	flag.Parse()
}

// Run 执行命令行参数对应的操作逻辑
func Run() {
	// 如果指定了-db参数，则执行数据库迁移并退出程序
	if Options.DB {
		Migrate()  // 执行数据库表结构迁移操作
		os.Exit(0) // 迁移完成后正常退出程序
	}
	// 如果指定了-v参数，则打印版本信息并退出程序
	if Options.Version {
		logrus.Infof("当前版本: %s  commit: %s, buildTime: %s",
			global.Version, global.Commit, global.BuildTime)
		os.Exit(0)
	}
}
