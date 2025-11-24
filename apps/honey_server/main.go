package main

import (
	"ThreatTrapMatrix/apps/honey_server/core"
	"ThreatTrapMatrix/apps/honey_server/flags"
	"ThreatTrapMatrix/apps/honey_server/global"
)

func main() {
	global.Config = core.ReadConfig()    // 读取配置文件
	global.Log = core.GetLogger()        // 获取日志实例
	global.DB = core.GetDB()             // 获取MySQL数据库实例
	global.Redis = core.GetRedisClient() // 获取Redis实例
	flags.Run()                          // 运行命令行参数
	global.Log.Infof("info日志")
	global.Log.Warnf("warn日志")
	global.Log.Errorf("error日志")
}
