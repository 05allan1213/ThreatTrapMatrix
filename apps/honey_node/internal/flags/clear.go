package flags

// File: honey_node/flags/clear.go
// Description: 节点数据清理模块，提供节点全量数据重置能力，清空IP、端口、任务数据库记录的同时清理关联的网络接口资源

import (
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/ip_service"

	"github.com/sirupsen/logrus"
)

// Clear 清空节点的全量业务数据
func Clear() {
	// 1. 清理IP相关资源及记录：先删除IP关联的网络接口，再删除IP数据库记录
	var ipList []models.IpModel
	global.DB.Find(&ipList) // 查询所有IP记录
	for _, model := range ipList {
		ip_service.RemoveInterface(model.LinkName) // 移除IP关联的网络接口，释放网络资源
	}
	// 批量删除IP数据库记录（仅当有记录时执行）
	if len(ipList) > 0 {
		global.DB.Delete(&ipList)
	}
	logrus.Infof("删除ip记录%d条", len(ipList))

	// 2. 清理端口数据库记录
	var portList []models.PortModel
	global.DB.Find(&portList) // 查询所有端口记录
	if len(portList) > 0 {
		global.DB.Delete(&portList) // 批量删除端口记录
	}
	logrus.Infof("删除端口记录%d条", len(portList))

	// 3. 清理任务数据库记录
	var taskList []models.TaskModel
	global.DB.Find(&taskList) // 查询所有任务记录
	if len(taskList) > 0 {
		global.DB.Delete(&taskList) // 批量删除任务记录
	}
	logrus.Infof("删除任务记录%d条", len(taskList))
}
