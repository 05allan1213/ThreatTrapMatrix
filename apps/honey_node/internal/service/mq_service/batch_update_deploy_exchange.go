package mq_service

// File: honey_node/service/mq_service/batch_update_deploy_exchange.go
// Description: 实现节点侧批量更新部署MQ消息消费逻辑，包含任务入库、异步清理旧端口转发、重建新端口转发、更新状态上报及任务状态更新等核心功能

import (
	"encoding/json"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/port_service"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BatchUpdateDeployExChange 节点侧批量更新部署MQ消息处理入口函数
func BatchUpdateDeployExChange(msg string) error {
	// 解析MQ消息体为批量更新部署请求结构体
	var req models.BatchUpdateDeployRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		logrus.Errorf("JSON解析失败: %v, 消息: %s", err, msg)
		return nil // 保持原有逻辑，解析失败时返回nil，不中断后续处理
	}

	// 生成唯一任务ID，用于标识本次批量更新部署任务
	taskID := uuid.New().String()
	// 将批量更新部署任务入库，记录任务基础信息及原始更新指令
	err := global.DB.Create(&models.TaskModel{
		TaskID:                taskID, // 任务唯一标识
		Type:                  2,      // 任务类型：2-批量更新部署
		BatchUpdateDeployData: &req,   // 批量更新部署原始指令数据
		Status:                0,      // 任务状态：0-执行中
	}).Error
	if err != nil {
		logrus.Errorf("任务入库失败 %s", err)
		return err
	}

	// 异步执行批量更新部署核心逻辑（非阻塞，避免阻塞MQ消费流程）
	go UpdateDeployTask(req, taskID)
	return nil
}

// UpdateDeployTask 批量更新部署核心任务执行函数
func UpdateDeployTask(req models.BatchUpdateDeployRequest, taskID string) {
	// 第一步：清理待更新IP的所有旧端口转发（先删后建，保证配置更新的准确性）
	for _, s := range req.IpList {
		port_service.CloseIpTunnel(s) // 关闭指定IP的所有端口隧道
	}

	// 第二步：按IP分组构建新端口转发配置映射表（便于按IP批量处理）
	var ipPortMap = map[string][]models.PortInfo{}
	for _, info := range req.PortList {
		ipPortMap[info.IP] = append(ipPortMap[info.IP], info)
	}

	// 第三步：按IP批量重建新端口转发，并上报更新状态
	for ip, portList := range ipPortMap {
		// 初始化更新状态上报数据（基础字段）
		res := UpdateDeployStatusRequest{
			NetID:    req.NetID,
			IP:       ip,
			LogID:    req.LogID,
			ErrorMsg: "",
		}

		// 为当前IP创建所有新端口转发
		for _, info := range portList {
			// 建立端口隧道（本地地址→目标地址）
			err := port_service.Tunnel(info.LocalAddr(), info.TargetAddr())
			// 初始化端口状态上报数据
			pI := PortInfo{
				Port: info.Port,
			}

			// 端口隧道创建失败时记录错误信息
			if err != nil {
				pI.ErrorMsg = err.Error()
			} else {
				// 端口隧道创建成功，持久化端口配置到数据库
				global.DB.Create(&models.PortModel{
					TargetAddr: info.TargetAddr(),
					LocalAddr:  info.LocalAddr(),
				})
			}
			// 将当前端口的状态添加到IP的上报列表中
			res.PortList = append(res.PortList, pI)
		}

		// 上报当前IP的更新部署状态（包含所有端口的执行结果）
		SendUpdateDeployStatusMsg(res)
	}

	logrus.Infof("批量更新部署结束")

	// 第四步：更新批量更新部署任务状态为执行完成
	var taskModel models.TaskModel
	err := global.DB.Take(&taskModel, "task_id = ?", taskID).Error
	if err != nil {
		logrus.Errorf("%s 任务不存在", taskID)
		return
	}
	// 更新任务状态为1（执行完成）
	global.DB.Model(&taskModel).Update("status", 1)
}
