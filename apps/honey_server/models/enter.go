package models

// File: honey_server/models/enter.go
// Description: 数据基础模型定义，提供通用基础模型和分页信息结构体

import (
	"time"

	"gorm.io/gorm"
)

// Model 通用基础模型结构体
// 包含主键、创建时间、更新时间、软删除字段，作为所有业务模型的嵌入基类
type Model struct {
	ID        uint           `gorm:"primarykey" json:"id"`   // 主键ID（自增）
	CreatedAt time.Time      `json:"createdAt"`              // 记录创建时间
	UpdatedAt time.Time      `json:"updatedAt"`              // 记录最后更新时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"` // 软删除标识（gorm内置）
}

// PageInfo 分页查询参数结构体
// 用于接收前端传递的分页、搜索相关参数
type PageInfo struct {
	Page  int    `form:"page"`  // 当前页码（默认第1页）
	Limit int    `form:"limit"` // 每页记录数（默认10条）
	Key   string `form:"key"`   // 全局搜索关键词（用于模糊查询）
}
