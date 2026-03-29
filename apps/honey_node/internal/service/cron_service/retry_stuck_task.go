package cron_service

// File: honey_node/service/cron_service/retry_stuck_task.go
// Description: 批量任务运行期自愈任务实现，负责周期性扫描本地未完成任务并触发检查点恢复，避免任务卡死必须依赖节点重启

import "honey_node/internal/service/task_service"

// RetryStuckTask 定时扫描并恢复本地未完成的批量任务
func RetryStuckTask() {
	task_service.Run()
}
