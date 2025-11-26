package main

import (
	core2 "ThreatTrapMatrix/apps/honey_server/internal/core"
	"ThreatTrapMatrix/apps/honey_server/internal/flags"
	"ThreatTrapMatrix/apps/honey_server/internal/global"
	"ThreatTrapMatrix/apps/honey_server/internal/routers"
)

func main() {
	core2.InitIPDB()                      // 初始化IP地址数据库
	global.Config = core2.ReadConfig()    // 读取配置文件
	core2.SetLogDefault()                 // 设置默认日志配置
	global.Log = core2.GetLogger()        // 获取日志实例
	global.DB = core2.GetDB()             // 获取MySQL数据库实例
	global.Redis = core2.GetRedisClient() // 获取Redis实例
	flags.Run()                           // 运行命令行参数
	routers.Run()                         // 启动路由
}
