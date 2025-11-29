package cron_service

// File: honey_node/service/cron_service/enter.go
// Description: 定时任务服务管理模块，基于robfig/cron实现秒级定时任务调度，负责周期性执行系统资源采集与上报任务

import (
	"time"

	"github.com/robfig/cron/v3"
)

// Run 启动定时任务服务
func Run() {
	// 加载上海时区（东八区），确保定时任务按本地时区执行
	timezone, _ := time.LoadLocation("Asia/Shanghai")

	// 创建Cron实例：启用秒级精度（默认是分钟级），并指定时区
	crontab := cron.New(
		cron.WithSeconds(),          // 支持秒级定时表达式（格式：秒 分 时 日 月 周）
		cron.WithLocation(timezone), // 设置定时任务的时区
	)

	// 添加定时任务：每5秒执行一次Resource函数（资源采集与上报）
	crontab.AddFunc("*/5 * * * * *", Resource)

	// 启动Cron调度器（非阻塞，后台运行）
	crontab.Start()
}
