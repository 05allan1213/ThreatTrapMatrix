package models

import (
	"time"
)

// Model 模型基类
type Model struct {
	ID        uint      `gorm:"primaryKey" json:"id"` // 主键ID
	CreatedAt time.Time `json:"createdAt"`            // 创建时间
	UpdatedAt time.Time `json:"updatedAt"`            // 更新时间
}
