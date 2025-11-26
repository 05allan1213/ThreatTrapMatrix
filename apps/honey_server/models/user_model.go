package models

import "gorm.io/gorm"

// UserModel 用户模型
type UserModel struct {
	gorm.Model
	Username      string `gorm:"size:32" json:"username"`      // 用户名
	Role          int8   `json:"role"`                         // 角色
	Password      string `gorm:"size:64" json:"-"`             // 密码
	LastLoginDate string `gorm:"size:32" json:"lastLoginDate"` // 最后登录时间
}
