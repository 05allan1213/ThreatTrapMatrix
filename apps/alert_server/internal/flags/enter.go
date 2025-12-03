package flags

// File: alert_server/flag/enter.go
// Description: 命令行参数解析模块，处理命令行参数解析及对应功能调度

import (
	"alert_server/internal/global"
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

// FlagOptions 命令行参数配置结构体，存储解析后的所有命令行参数
type FlagOptions struct {
	File    string // 配置文件路径参数
	Version bool   // 版本信息打印开关
	DB      bool   // 数据库表结构迁移开关
	ES      bool   // es索引创建开关
	Menu    string // 功能菜单参数
	Type    string // 功能子类型参数
	Value   string // 功能参数值
	Help    bool   // 帮助信息展示开关
}

// Options 全局命令行参数实例，存储解析后的参数数据
var Options FlagOptions

// init 初始化命令行参数解析，注册参数选项并初始化命令注册
func init() {
	// 注册命令行参数及默认值、说明
	flag.StringVar(&Options.File, "f", "settings.yaml", "配置文件路径")
	flag.BoolVar(&Options.Version, "vv", false, "打印当前版本")
	flag.BoolVar(&Options.Help, "h", false, "帮助信息")
	flag.BoolVar(&Options.DB, "db", false, "迁移表结构")
	flag.BoolVar(&Options.ES, "es", false, "创建es索引")
	flag.StringVar(&Options.Menu, "m", "", "菜单 user")
	flag.StringVar(&Options.Type, "t", "", "类型 create list")
	flag.StringVar(&Options.Value, "v", "", "值")
	flag.Parse() // 解析命令行参数

	// 注册业务命令
	RegisterCommand()
}

// RegisterCommand 注册业务相关命令
func RegisterCommand() {

}

// runBaseCommand 执行基础命令
func runBaseCommand() {
	// 数据库表结构迁移命令
	if Options.DB {
		Migrate()
		os.Exit(0)
	}
	// es索引创建命令
	if Options.ES {
		EsIndex()
		os.Exit(0)
	}
	// 版本信息打印命令
	if Options.Version {
		logrus.Infof("当前版本: %s  commit: %s, buildTime: %s",
			global.Version, global.Commit, global.BuildTime)
		os.Exit(0)
	}
}

// runHelpCommand 处理帮助信息展示逻辑
// 支持全局帮助、菜单级帮助两种模式
func runHelpCommand() {
	// 全局帮助（无菜单参数时）
	if Options.Menu == "" && Options.Type == "" && Options.Help {
		fmt.Printf("菜单项:\n")
		for key := range HelpCommandMap {
			fmt.Printf("%s 使用 -m %s -h 查看具体子菜单\n", key, key)
		}
		os.Exit(0)
	}
	// 菜单级帮助（指定菜单时）
	if Options.Menu != "" && Options.Type == "" && Options.Help {
		subMenuMap, ok := HelpCommandMap[Options.Menu]
		if !ok {
			logrus.Fatalf("不存在的菜单项 %s", Options.Menu)
		}
		for key, help := range subMenuMap {
			fmt.Printf("%s %s\n", key, help)
		}
		os.Exit(0)
	}
}

// runCommand 执行已注册的业务命令
func runCommand() {
	// 菜单或子类型参数为空时直接返回
	if Options.Menu == "" || Options.Type == "" {
		return
	}
	// 构建命令唯一标识（菜单:子类型）
	key := fmt.Sprintf("%s:%s", Options.Menu, Options.Type)
	command, ok := CommandMap[key]
	if !ok {
		logrus.Fatalf("不存在的菜单项 %s %s", Options.Menu, Options.Type)
	}
	// 执行命令函数
	command.Func()
	os.Exit(0)
}

// Command 命令结构体，封装命令的菜单、子类型、帮助信息及执行函数
type Command struct {
	Menu string // 所属菜单
	Type string // 子类型
	Help string // 帮助描述信息
	Func func() // 命令执行函数
}

// CommandMap 命令注册表，以"菜单:子类型"为键存储命令实例
var CommandMap = map[string]*Command{}

// HelpCommandMap 帮助信息注册表，存储各菜单的子命令帮助信息
var HelpCommandMap = map[string]map[string]string{}

// registerCommand 注册命令到全局注册表
func registerCommand(menu, subMenu, help string, fun func()) {
	key := fmt.Sprintf("%s:%s", menu, subMenu)
	// 注册命令到CommandMap
	CommandMap[key] = &Command{
		Menu: menu,
		Type: subMenu,
		Help: help,
		Func: fun,
	}
	// 注册帮助信息到HelpCommandMap
	subMenuMap, ok := HelpCommandMap[menu]
	if ok {
		subMenuMap[subMenu] = help
	} else {
		HelpCommandMap[menu] = map[string]string{
			subMenu: help,
		}
	}
}

// Run 命令行入口函数，依次执行基础命令、帮助命令、业务命令
func Run() {
	// 运行基础命令（数据库迁移、版本查询）
	runBaseCommand()
	// 运行帮助的命令（全局帮助、菜单级帮助）
	runHelpCommand()
	// 运行注册的业务命令
	runCommand()
}
