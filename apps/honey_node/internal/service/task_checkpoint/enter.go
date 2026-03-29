package task_checkpoint

// File: honey_node/service/task_checkpoint/enter.go
// Description: 任务检查点服务模块，负责批量任务逐IP检查点的串行化读写、老数据兼容补建以及任务完成态收敛

import (
	"fmt"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"sync"
)

// taskLockerMap 按TaskID维护进程内互斥锁，避免并发覆盖同一行检查点
var taskLockerMap sync.Map

// getTaskLocker 获取指定任务的进程内互斥锁
func getTaskLocker(taskID string) *sync.Mutex {
	locker, _ := taskLockerMap.LoadOrStore(taskID, &sync.Mutex{})
	return locker.(*sync.Mutex)
}

// GetTask 获取任务并在必要时补建老数据检查点
func GetTask(taskID string) (task models.TaskModel, err error) {
	locker := getTaskLocker(taskID)
	locker.Lock()
	defer locker.Unlock()

	err = global.DB.Take(&task, "task_id = ?", taskID).Error
	if err != nil {
		return
	}

	// 老任务没有逐IP检查点时，启动恢复前先补建一份，避免任务永久卡死
	if task.EnsureTaskItems() {
		err = global.DB.Save(&task).Error
	}
	return
}

// UpdateTask 串行更新指定任务的检查点内容
func UpdateTask(taskID string, updater func(task *models.TaskModel) error) error {
	locker := getTaskLocker(taskID)
	locker.Lock()
	defer locker.Unlock()

	var task models.TaskModel
	if err := global.DB.Take(&task, "task_id = ?", taskID).Error; err != nil {
		return err
	}

	// 每次更新前都保证老任务已经补齐检查点
	task.EnsureTaskItems()
	if err := updater(&task); err != nil {
		return err
	}
	return global.DB.Save(&task).Error
}

// CompleteTaskIfReported 当所有逐IP状态都回传完成后，将任务标记为完成
func CompleteTaskIfReported(taskID string) error {
	return UpdateTask(taskID, func(task *models.TaskModel) error {
		if task.AllTaskItemsReported() {
			task.Status = models.TaskStatusCompleted
		}
		return nil
	})
}

// BuildMissingTaskDataError 构造任务原始数据缺失时的统一错误
func BuildMissingTaskDataError(taskID string) error {
	return fmt.Errorf("task %s missing original task data", taskID)
}
