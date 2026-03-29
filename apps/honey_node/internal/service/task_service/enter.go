package task_service

// File: honey_node/service/task_service/enter.go
// Description: 节点侧任务恢复服务模块，负责扫描本地未完成任务，并按任务类型恢复批量部署、批量更新和批量删除

import (
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/mq_service"

	"github.com/sirupsen/logrus"
)

// Run 恢复执行节点侧所有未完成任务
func Run() {
	// 查询数据库中所有状态为运行中的任务记录
	var taskList []models.TaskModel
	global.DB.Find(&taskList, "status = ?", models.TaskStatusRunning)

	if len(taskList) > 0 {
		logrus.Warnf("存在%d个未完成的任务", len(taskList))
	}

	// 按任务类型分发恢复逻辑，避免只有部署任务能续跑
	for _, task := range taskList {
		switch task.Type {
		case models.TaskTypeBatchDeploy:
			logrus.Infof("恢复未完成的批量部署任务 %s", task.TaskID)
			if err := mq_service.ResumeDeployTask(task.TaskID); err != nil {
				logrus.Errorf("恢复批量部署任务失败 %s %v", task.TaskID, err)
			}
		case models.TaskTypeBatchUpdate:
			logrus.Infof("恢复未完成的批量更新任务 %s", task.TaskID)
			if err := mq_service.ResumeUpdateTask(task.TaskID); err != nil {
				logrus.Errorf("恢复批量更新任务失败 %s %v", task.TaskID, err)
			}
		case models.TaskTypeBatchRemove:
			logrus.Infof("恢复未完成的批量删除任务 %s", task.TaskID)
			if err := mq_service.ResumeRemoveTask(task.TaskID); err != nil {
				logrus.Errorf("恢复批量删除任务失败 %s %v", task.TaskID, err)
			}
		default:
			logrus.Warnf("未识别的任务类型，跳过恢复 taskID=%s type=%d", task.TaskID, task.Type)
		}
	}
}
