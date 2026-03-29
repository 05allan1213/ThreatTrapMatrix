package mq_service

// File: honey_node/service/mq_service/batch_remove_deploy_exchange.go
// Description: 节点侧批量删除部署MQ消息消费模块，负责任务入库、逐IP资源清理、状态回传以及重启恢复所需的检查点初始化

import (
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/task_checkpoint"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BatchRemoveDeployExChange 处理批量删除部署的MQ消息
func BatchRemoveDeployExChange(req models.BatchRemoveDeployRequest) error {
	// 生成唯一任务ID，用于标识本次批量删除任务
	taskID := uuid.New().String()

	// 先将任务落到本地SQLite，并初始化逐IP检查点，便于重启后继续补动作和补回调
	err := global.DB.Create(&models.TaskModel{
		TaskID:                taskID,
		Type:                  models.TaskTypeBatchRemove,
		BatchRemoveDeployData: &req,
		RemoveItems:           models.BuildRemoveTaskItems(&req),
		Status:                models.TaskStatusRunning,
	}).Error
	if err != nil {
		logrus.Errorf("任务入库失败 %s", err)
		return err
	}

	// 异步执行批量删除，避免阻塞MQ消费流程
	go RemoveDeployTask(req, taskID)
	return nil
}

// RemoveDeployTask 批量删除部署核心任务执行函数
func RemoveDeployTask(req models.BatchRemoveDeployRequest, taskID string) {
	log := core.GetLogger().WithField("logID", req.LogID)
	log.WithField("data", req).Info("批量删除部署开始")

	// 同一个任务同一时刻只允许一个执行流进入，避免和启动恢复、定时自愈并发撞车
	if !task_checkpoint.TryLockTaskExecution(taskID) {
		log.WithField("task_id", taskID).Warn("批量删除任务正在执行中，跳过本次执行")
		return
	}
	defer task_checkpoint.UnlockTaskExecution(taskID)

	// 逐IP执行删除，并推进检查点
	for _, info := range req.IPList {
		if err := processRemoveTaskItem(taskID, req, info.Ip, info.LinkName, log); err != nil {
			log.WithError(err).WithField("ip", info.Ip).Error("批量删除执行失败")
		}
	}

	log.Info("批量删除部署结束")

	// 只有所有逐IP状态都成功回传后，整批任务才会收敛为完成
	if err := task_checkpoint.CompleteTaskIfReported(taskID); err != nil {
		logrus.Errorf("%s 删除任务完成状态收敛失败: %v", taskID, err)
	}
}
