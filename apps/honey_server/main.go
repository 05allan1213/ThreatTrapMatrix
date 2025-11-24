package main

import (
	"ThreatTrapMatrix/apps/honey_server/core"
	"ThreatTrapMatrix/apps/honey_server/flags"
	"ThreatTrapMatrix/apps/honey_server/global"
)

func main() {
	global.DB = core.InitDB() // 初始化MySQL数据库
	flags.Run()               // 运行命令行参数
}
