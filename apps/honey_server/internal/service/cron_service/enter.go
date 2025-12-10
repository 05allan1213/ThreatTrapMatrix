package cron_service

// File: honey_server/service/cron_service/enter.go
// Description: 定时任务服务模块，初始化基于上海时区的定时任务调度器，注册虚拟服务端口同步定时任务并启动调度器

import (
	"time"

	"github.com/robfig/cron/v3"
)

// Run 启动定时任务调度器
func Run() {
	// 加载上海时区，确保定时任务按北京时间执行
	timezone, _ := time.LoadLocation("Asia/Shanghai")

	// 创建crontab实例：启用秒级调度精度，指定上海时区
	crontab := cron.New(cron.WithSeconds(), cron.WithLocation(timezone))

	// 注册定时任务：每分钟的0秒、每20分钟执行一次SyncVsServicePort函数
	crontab.AddFunc("0 */20 * * * *", SyncVsServicePort)

	// 启动定时任务调度器（非阻塞，后台运行）
	crontab.Start()
}
