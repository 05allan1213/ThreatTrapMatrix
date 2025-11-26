package main

import (
	"ThreatTrapMatrix/apps/honey_server/core"
	"ThreatTrapMatrix/apps/honey_server/flags"
	"ThreatTrapMatrix/apps/honey_server/global"
	"ThreatTrapMatrix/apps/honey_server/routers"
)

func main() {
	core.InitIPDB()                      // 初始化IP地址数据库
	global.Config = core.ReadConfig()    // 读取配置文件
	core.SetLogDefault()                 // 设置默认日志配置
	global.Log = core.GetLogger()        // 获取日志实例
	global.DB = core.GetDB()             // 获取MySQL数据库实例
	global.Redis = core.GetRedisClient() // 获取Redis实例
	flags.Run()                          // 运行命令行参数
	routers.Run()                        // 启动路由
}
