package models

import "gorm.io/gorm"

// UserModel 用户模型
type UserModel struct {
	gorm.Model
	Username      string `json:"username"`      // 用户名
	Role          string `json:"role"`          // 角色
	Password      string `json:"-"`             // 密码
	LastLoginDate string `json:"lastLoginDate"` // 最后登录时间
}
