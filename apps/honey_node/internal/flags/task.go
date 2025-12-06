package flags

// File: honey_node/flags/task.go
// Description: 提供任务相关的命令行标识（flag）操作，实现节点侧已执行任务列表的查询功能

import (
	"fmt"
	"honey_node/internal/global"
	"honey_node/internal/models"
)

// taskService 任务相关命令行操作服务结构体
type taskService struct{}

// List 查询并打印节点侧所有任务记录
func (taskService) List() {
	// 查询数据库中所有任务记录
	var list []models.TaskModel
	global.DB.Find(&list)
	// 遍历并格式化打印每条任务记录
	for _, model := range list {
		fmt.Printf("%#v\n", model)
	}
}
