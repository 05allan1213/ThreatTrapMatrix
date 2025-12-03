package main

import (
	"matrix_server/internal/core"
	"matrix_server/internal/flags"
	"matrix_server/internal/global"
	"matrix_server/internal/routers"
	"matrix_server/internal/service/mq_service"
)

func main() {
	core.InitIPDB()                      // 初始化IP地址数据库
	global.Config = core.ReadConfig()    // 读取配置文件
	core.SetLogDefault()                 // 设置默认日志配置
	global.Log = core.GetLogger()        // 获取日志实例
	global.DB = core.GetDB()             // 获取MySQL数据库实例
	global.Redis = core.GetRedisClient() // 获取Redis实例
	global.Queue = core.InitMQ()         // 初始化消息队列
	mq_service.RegisterExChange()        // 注册交换机
	flags.Run()                          // 运行命令行参数
	routers.Run()                        // 启动路由
}
