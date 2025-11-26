package flags

// File: honey_server/flag/enter.go
// Description: 命令行参数解析模块，处理命令行参数解析及对应功能调度

import (
	"flag"
	"os"

	"ThreatTrapMatrix/apps/honey_server/global"

	"github.com/sirupsen/logrus"
)

// FlagOptions 命令行参数配置结构体，存储解析后的命令行参数值
type FlagOptions struct {
	File    string // 配置文件路径参数
	Version bool   // 版本信息打印开关
	DB      bool   // 数据库表结构迁移开关
	Menu    string // 功能菜单参数
	Type    string // 功能类型参数
	Value   string // 功能参数值
}

// Options 全局命令行参数实例，存储解析后的参数数据
var Options FlagOptions

// init 初始化命令行参数解析，注册各参数选项及默认值
func init() {
	flag.StringVar(&Options.File, "f", "settings.yaml", "配置文件路径")
	flag.BoolVar(&Options.Version, "vv", false, "打印当前版本")
	flag.BoolVar(&Options.DB, "db", false, "迁移表结构")
	flag.StringVar(&Options.Menu, "m", "", "菜单 user")
	flag.StringVar(&Options.Type, "t", "", "类型 create list")
	flag.StringVar(&Options.Value, "v", "", "值")
	flag.Parse() // 解析命令行参数
}

// Run 根据解析后的命令行参数调度对应功能执行
func Run() {
	// 数据库表结构迁移功能
	if Options.DB {
		Migrate()
		os.Exit(0)
	}

	// 版本信息打印功能
	if Options.Version {
		logrus.Infof("当前版本: %s  commit: %s, buildTime: %s",
			global.Version, global.Commit, global.BuildTime)
		os.Exit(0)
	}

	// 功能菜单分支处理
	switch Options.Menu {
	case "user":
		var user User
		// 用户菜单下的子功能分支
		switch Options.Type {
		case "create":
			user.Create(Options.Value)
			os.Exit(0)
		case "list":
			user.List()
			os.Exit(0)
		default:
			logrus.Fatalf("用户子菜单项不正确")
		}
	case "":
		// 无菜单参数时的默认处理（空实现）
	default:
		logrus.Fatalf("菜单项不正确")
	}
}
