package mq_service

// File: honey_node/service/mq_service/batch_deploy_exchange.go
// Description: 节点侧批量部署MQ消息消费模块，负责任务入库、逐IP诱捕资源部署、端口规则补齐以及重启恢复所需的检查点初始化

import (
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/task_checkpoint"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BatchDeployExChange 节点侧批量部署MQ消息处理函数
func BatchDeployExChange(req models.BatchDeployRequest) error {
	// 生成唯一任务ID，用于标识本次批量部署任务
	taskID := uuid.New().String()

	// 先将任务落到本地SQLite，并初始化逐IP检查点，便于重启后继续补动作和补回调
	err := global.DB.Create(&models.TaskModel{
		TaskID:          taskID,
		Type:            models.TaskTypeBatchDeploy,
		BatchDeployData: &req,
		DeployItems:     models.BuildDeployTaskItems(&req),
		Status:          models.TaskStatusRunning,
	}).Error
	if err != nil {
		logrus.Errorf("任务入库失败 %s", err)
		return err
	}

	// 异步执行批量部署，避免阻塞MQ消费流程
	go DeployTask(req, taskID)
	return nil
}

// DeployTask 批量部署核心任务执行函数
func DeployTask(req models.BatchDeployRequest, taskID string) {
	log := core.GetLogger().WithField("logID", req.LogID)
	log.WithField("data", req).Info("节点开始部署")

	// 控制单批部署并发，避免一次性创建过多网卡和ARP探测把节点打满
	maxChan := make(chan struct{}, 200)
	var wait sync.WaitGroup

	for _, info := range req.IPList {
		maxChan <- struct{}{}
		wait.Add(1)

		go func(deployInfo models.DeployIp) {
			defer func() {
				<-maxChan
				wait.Done()
			}()

			// 逐IP执行部署，并推进检查点
			if err := processDeployTaskItem(taskID, req, deployInfo); err != nil {
				log.WithError(err).WithField("ip", deployInfo.Ip).Error("批量部署执行失败")
			}
		}(info)
	}

	wait.Wait()

	// 批量部署完成后补齐缺失的端口规则，避免“IP已存在但端口转发丢失”
	reconcileDeployTaskPorts(req, log)

	log.WithField("count", len(req.IPList)).Info("批量部署结束")

	// 只有所有逐IP状态都成功回传后，整批任务才会收敛为完成
	if err := task_checkpoint.CompleteTaskIfReported(taskID); err != nil {
		logrus.Errorf("%s 部署任务完成状态收敛失败: %v", taskID, err)
	}
}
