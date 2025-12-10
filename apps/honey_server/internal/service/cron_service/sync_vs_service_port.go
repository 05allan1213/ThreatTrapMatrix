package cron_service

// File: honey_server/service/cron_service/sync_vs_service_port.go
// Description: 定时任务服务模块，提供虚拟服务端口数同步功能，统计端口转发表中各服务关联的端口数量，并更新到虚拟服务表的端口数字段

import (
	"honey_server/internal/global"
	"honey_server/internal/models"

	"github.com/sirupsen/logrus"
)

// VsTable 端口转发按服务ID聚合的统计结果结构体
type VsTable struct {
	ServiceID uint `json:"serviceID" gorm:"column:service_id"` // 虚拟服务ID
	Count     int  `json:"count" gorm:"column:count"`          // 该服务下关联的端口转发总数
}

// SyncVsServicePort 同步虚拟服务关联的端口转发数量
func SyncVsServicePort() {
	// 定义切片接收按服务ID分组的端口统计结果
	var tableList []VsTable
	// 从端口转发表分组查询：按service_id聚合，统计每个服务下的端口总数
	global.DB.Model(models.HoneyPortModel{}).
		Group("service_id"). // 按服务ID分组
		Select("service_id", "count(id) as count"). // 选择服务ID和端口数量（count(id)统计端口数）
		Scan(&tableList) // 将查询结果扫描到tableList

	// 将统计结果转为map，便于快速查询指定服务ID的端口数
	var tableMap = map[uint]int{}
	for _, table := range tableList {
		tableMap[table.ServiceID] = table.Count
	}

	// 查询所有虚拟服务记录
	var serviceList []models.ServiceModel
	global.DB.Find(&serviceList)

	// 遍历所有虚拟服务，对比并更新端口数
	for _, model := range serviceList {
		// 从统计map中获取当前服务的端口总数
		count, ok := tableMap[model.ID]
		if ok {
			// 服务存在关联端口，且当前记录的端口数与实际统计数不一致时更新
			if model.HoneyIPCount != count {
				logrus.Infof("更新虚拟服务 %s 的端口数 绑定端口数 %d -> %d",
					model.Title, model.HoneyIPCount, count)
				global.DB.Model(&model).Update("honey_ip_count", count)
			}
			continue // 处理下一个服务
		}

		// 服务无关联端口，但当前记录的端口数非0时，更新为0
		if model.HoneyIPCount != 0 {
			logrus.Infof("更新虚拟服务 %s 的端口数 绑定端口数 %d -> %d",
				model.Title, model.HoneyIPCount, 0)
			global.DB.Model(&model).Update("honey_ip_count", 0)
		}
	}
}
