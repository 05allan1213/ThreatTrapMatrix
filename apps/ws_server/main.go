package main

import (
	"ws_server/internal/core"
	"ws_server/internal/global"
	"ws_server/internal/routers"
	"ws_server/internal/service/mq_service"
)

func main() {
	core.InitIPDB()                   // 初始化IP地址数据库
	global.Config = core.ReadConfig() // 读取配置文件
	core.SetLogDefault()              // 设置默认日志配置
	global.Log = core.GetLogger()     // 获取日志实例
	global.Queue = core.InitMQ()      // 初始化消息队列
	mq_service.Run()                  // 启动MQ服务
	routers.Run()                     // 启动路由
}
