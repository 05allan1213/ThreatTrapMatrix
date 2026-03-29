package mq_service

// File: honey_node/service/mq_service/batch_update_deploy_exchange.go
// Description: 节点侧批量更新部署MQ消息消费模块，负责任务入库、逐IP端口规则更新、状态回传以及重启恢复所需的检查点初始化

import (
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/task_checkpoint"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BatchUpdateDeployExChange 节点侧批量更新部署MQ消息处理入口
func BatchUpdateDeployExChange(req models.BatchUpdateDeployRequest) error {
	// 生成唯一任务ID，用于标识本次批量更新部署任务
	taskID := uuid.New().String()

	// 先将任务落到本地SQLite，并初始化逐IP检查点，便于重启后继续补动作和补回调
	err := global.DB.Create(&models.TaskModel{
		TaskID:                taskID,
		Type:                  models.TaskTypeBatchUpdate,
		BatchUpdateDeployData: &req,
		UpdateItems:           models.BuildUpdateTaskItems(&req),
		Status:                models.TaskStatusRunning,
	}).Error
	if err != nil {
		logrus.Errorf("任务入库失败 %s", err)
		return err
	}

	// 异步执行批量更新，避免阻塞MQ消费流程
	go UpdateDeployTask(req, taskID)
	return nil
}

// UpdateDeployTask 批量更新部署核心任务执行函数
func UpdateDeployTask(req models.BatchUpdateDeployRequest, taskID string) {
	log := core.GetLogger().WithField("logID", req.LogID)
	log.WithField("data", req).Info("批量更新部署")

	// 同一个任务同一时刻只允许一个执行流进入，避免和启动恢复、定时自愈并发撞车
	if !task_checkpoint.TryLockTaskExecution(taskID) {
		log.WithField("task_id", taskID).Warn("批量更新任务正在执行中，跳过本次执行")
		return
	}
	defer task_checkpoint.UnlockTaskExecution(taskID)

	// 按IP分组构建目标端口规则，逐IP执行并推进检查点
	ipPortMap := buildUpdatePortMap(req)
	for _, ip := range req.IpList {
		if err := processUpdateTaskItem(taskID, req, ip, ipPortMap[ip], log); err != nil {
			log.WithError(err).WithField("ip", ip).Error("批量更新执行失败")
		}
	}

	log.WithField("ipCount", len(ipPortMap)).Info("批量更新部署结束")

	// 只有所有逐IP状态都成功回传后，整批任务才会收敛为完成
	if err := task_checkpoint.CompleteTaskIfReported(taskID); err != nil {
		logrus.Errorf("%s 更新任务完成状态收敛失败: %v", taskID, err)
	}
}
