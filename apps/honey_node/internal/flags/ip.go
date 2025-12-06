package flags

// File: honey_node/flags/ip.go
// Description: 提供IP相关的命令行标识（flag）操作，实现节点侧已配置诱捕IP列表的查询功能

import (
	"fmt"
	"honey_node/internal/global"
	"honey_node/internal/models"
)

// ipService IP相关命令行操作服务结构体
type ipService struct{}

// List 查询并打印节点侧所有已配置的诱捕IP信息
func (ipService) List() {
	// 查询数据库中所有诱捕IP配置记录
	var list []models.IpModel
	global.DB.Find(&list)
	// 遍历并格式化打印每条IP配置记录
	for _, model := range list {
		fmt.Printf("%#v\n", model)
	}
}
