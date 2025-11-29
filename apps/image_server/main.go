package main

import (
	"image_server/internal/core"
	"image_server/internal/flags"
	"image_server/internal/global"
	"image_server/internal/routers"
	"image_server/internal/service/cron_service"
	"image_server/internal/service/vs_net_service"
)

func main() {
	core.InitIPDB()                         // 初始化IP地址数据库
	global.Config = core.ReadConfig()       // 读取配置文件
	core.SetLogDefault()                    // 设置默认日志配置
	global.DockerClient = core.InitDocker() // 初始化Docker客户端
	global.Log = core.GetLogger()           // 获取日志实例
	global.DB = core.GetDB()                // 获取MySQL数据库实例
	global.Redis = core.GetRedisClient()    // 获取Redis实例
	flags.Run()                             // 运行命令行参数
	cron_service.Run()                      // 运行定时任务
	vs_net_service.Run()                    // 运行虚拟网络服务
	routers.Run()                           // 启动路由
}
