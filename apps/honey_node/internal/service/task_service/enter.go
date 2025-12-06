package task_service

// File: honey_node/service/task_service/enter.go
// Description: 实现节点侧未完成任务的恢复执行功能，检测并重新执行状态为0的批量部署任务，筛选未部署的IP重新执行配置，保证部署任务的完整性和幂等性

import (
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/mq_service"

	"github.com/sirupsen/logrus"
)

// Run 恢复执行节点侧所有未完成的任务
func Run() {
	// 查询数据库中所有状态为0（执行中）的任务记录
	var taskList []models.TaskModel
	global.DB.Find(&taskList, "status = 0")
	// 日志提示未完成任务数量，便于排查
	if len(taskList) > 0 {
		logrus.Warnf("存在%d个未完成的任务", len(taskList))
	}

	// 遍历所有未完成任务，按任务类型处理
	for _, model := range taskList {
		// 仅处理批量部署任务（Type=1）
		if model.Type == 1 {
			logrus.Infof("未完成的批量部署任务 %s %#v", model.CreatedAt, model.BatchDeployData)

			// 第一步：查询该批量部署任务对应网络下已部署的IP列表
			var ipList []models.IpModel
			global.DB.Find(&ipList, "network = ?", model.BatchDeployData.Network)

			// 构建已部署IP的映射表，用于快速筛选未部署IP
			var deployIP = map[string]bool{}
			for _, ipModel := range ipList {
				deployIP[ipModel.Ip] = true
			}

			// 初始化未部署IP的批量部署请求结构体（复用原任务的子网、日志、网络信息）
			var unDeployData = models.BatchDeployRequest{
				NetID:   model.BatchDeployData.NetID,
				LogID:   model.BatchDeployData.LogID,
				Network: model.BatchDeployData.Network,
			}

			// 第二步：筛选原任务中未部署的IP，加入待重新部署列表
			for _, ip := range model.BatchDeployData.IPList {
				// 跳过已部署的IP，保证幂等性
				if deployIP[ip.Ip] {
					continue
				}
				logrus.Infof("未部署的ip %s", ip.Ip)
				// 将未部署IP加入待部署列表
				unDeployData.IPList = append(unDeployData.IPList, ip)
			}

			// 第三步：存在未部署IP则重新执行部署逻辑，否则更新任务状态为完成
			if len(unDeployData.IPList) > 0 {
				// 调用批量部署核心函数，执行未部署IP的配置（复用原任务ID）
				mq_service.DeployTask(unDeployData, model.TaskID)
			} else {
				// 无未部署IP，标记任务为完成（状态1）
				global.DB.Model(&model).Update("status", 1)
			}
		}
	}
}
