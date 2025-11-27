package cron_service

// File: image_server/service/cron_service/vs_health.go
// Description: 虚拟服务健康检查定时任务，同步Docker容器状态与数据库中服务状态

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/service/docker_service"
	"fmt"

	"github.com/sirupsen/logrus"
)

// VsHealth 虚拟服务健康检查核心逻辑
func VsHealth() {
	// 获取Docker引擎中所有容器的状态信息
	allContainers, err := docker_service.ListAllContainers()
	if err != nil {
		logrus.Errorf("容器状态检测失败 %s", err)
		return
	}

	// 查询数据库中所有虚拟服务记录
	var list []models.ServiceModel
	global.DB.Find(&list)
	// 构建容器ID到服务模型的映射（便于快速匹配）
	var containerMap = map[string]*models.ServiceModel{}
	for _, model := range list {
		containerMap[model.ContainerID] = &model
	}

	// 遍历所有Docker容器，对比服务状态
	for _, container := range allContainers {
		containerID := container.ID[:12] // 截取容器12位短ID（与数据库存储格式一致）
		model, ok := containerMap[containerID]
		if !ok {
			continue // 非当前系统管理的容器，跳过处理
		}

		var newModel models.ServiceModel
		var isUpdate bool

		// 场景1：数据库记录状态异常，但容器实际运行正常 → 同步为正常状态
		if container.State == "running" && model.Status != 1 {
			newModel.Status = 1
			newModel.ErrorMsg = ""
			isUpdate = true
		}

		// 场景2：数据库记录状态正常，但容器实际运行异常 → 同步为异常状态
		if container.State != "running" && model.Status == 1 {
			newModel.Status = 2
			newModel.ErrorMsg = fmt.Sprintf("%s(%s)", container.State, container.Status)
			isUpdate = true
		}

		// 存在状态差异时更新数据库
		if isUpdate {
			logrus.Infof("%s 容器存在状态修改 %s => %s", model.ContainerName, model.State(), newModel.State())
			global.DB.Model(model).Updates(map[string]any{
				"status":    newModel.Status,
				"error_msg": newModel.ErrorMsg,
			})
		}

		// 打印容器状态信息
		fmt.Printf("ID: %s, 名称: %s, 状态: %s\n", container.ID[:12], container.Names[0][1:], container.State)
	}
}
