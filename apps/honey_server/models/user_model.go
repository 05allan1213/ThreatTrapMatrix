package models

import (
	"gorm.io/gorm"
)

// UserModel 用户模型
type UserModel struct {
	Model
	Username      string `gorm:"size:32" json:"username"`      // 用户名
	Role          int8   `json:"role"`                         // 角色 1 管理员 2 普通用户
	Password      string `gorm:"size:64" json:"-"`             // 密码
	LastLoginDate string `gorm:"size:32" json:"lastLoginDate"` // 最后登录时间
}

func (UserModel) BeforeDelete(tx *gorm.DB) error {
	// fmt.Println("删除前")
	// return errors.New("删除失败")
	return nil
}
