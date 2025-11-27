package cron_service

// File: image_server/service/cron_service/enter.go
// Description: 定时任务服务管理，负责启动定时任务调度器并注册虚拟服务健康检查任务

import (
	"time"

	"github.com/robfig/cron/v3"
)

// Run 启动定时任务调度器，注册虚拟服务健康检查任务并启动调度器
func Run() {
	// 加载上海时区（东八区）
	timezone, _ := time.LoadLocation("Asia/Shanghai")
	// 创建定时任务实例，支持秒级精度并使用指定时区
	crontab := cron.New(cron.WithSeconds(), cron.WithLocation(timezone))
	// 注册虚拟服务健康检查任务（每秒执行一次）
	crontab.AddFunc("* * * * * *", VsHealth)
	// 启动定时任务调度器
	crontab.Start()
}
