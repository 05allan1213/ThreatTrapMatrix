package mq_service

// File: honey_node/service/mq_service/batch_remove_deploy_exchange.go
// Description: MQ消息消费处理模块，实现批量删除部署消息的解析、任务入库及异步执行删除部署任务，包含端口转发关闭、网络接口删除、任务状态更新等核心逻辑

import (
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/ip_service"
	"honey_node/internal/service/port_service"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BatchRemoveDeployExChange 处理批量删除部署的MQ消息
func BatchRemoveDeployExChange(req models.BatchRemoveDeployRequest) error {
	// 生成唯一任务ID
	taskID := uuid.New().String()
	// 创建批量删除部署任务记录并写入数据库
	err := global.DB.Create(&models.TaskModel{
		TaskID:                taskID, // 任务唯一标识
		Type:                  3,      // 任务类型：3表示批量删除部署
		BatchRemoveDeployData: &req,   // 批量删除部署任务关联的请求数据
		Status:                0,      // 任务状态：0表示待执行
	}).Error
	if err != nil {
		logrus.Errorf("任务入库失败 %s", err)
		return err
	}
	// 异步执行删除部署任务（非阻塞）
	go RemoveDeployTask(req, taskID)
	return nil
}

// RemoveDeployTask 执行批量删除部署的具体任务逻辑
func RemoveDeployTask(req models.BatchRemoveDeployRequest, taskID string) {
	// 遍历待删除部署的IP列表，逐个处理
	// 处理逻辑：1. 关闭端口转发 2. 删除网络接口 3. 发送单个IP删除部署状态消息
	for _, s := range req.IPList {
		// 关闭当前IP对应的端口转发
		port_service.CloseIpTunnel(s.Ip)

		// 组装IP删除部署状态请求数据
		res := RemoveDeployStatusRequest{
			NetID:    req.NetID,
			IP:       s.Ip,
			LogID:    req.LogID,
			ErrorMsg: "",
		}
		// 删除当前IP对应的网络接口
		err := ip_service.RemoveInterface(s.LinkName)
		if err != nil {
			// 接口删除失败时记录错误信息
			res.ErrorMsg = err.Error()
		}
		global.DB.Delete(&models.IpModel{}, "ip = ?", s.Ip)
		// 发送当前IP的删除部署状态消息
		SendRemoveDeployStatusMsg(res)
	}

	// 记录批量删除部署任务执行完成日志
	logrus.Infof("批量删除部署结束")

	// 查询任务记录并更新任务状态
	var taskModel models.TaskModel
	err := global.DB.Take(&taskModel, "task_id = ?", taskID).Error
	if err != nil {
		logrus.Errorf("%s 任务不存在", taskID)
		return
	}
	// 更新任务状态为已完成（1表示执行完成）
	global.DB.Model(&taskModel).Update("status", 1)
}
