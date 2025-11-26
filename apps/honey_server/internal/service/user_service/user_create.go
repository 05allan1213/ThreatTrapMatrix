package user_service

// File: honey_server/service/user_service/user_create.go
// Description: 用户服务模块，实现用户创建的核心业务逻辑处理

import (
	"ThreatTrapMatrix/apps/honey_server/internal/global"
	"ThreatTrapMatrix/apps/honey_server/internal/models"
	"ThreatTrapMatrix/apps/honey_server/internal/utils/pwd"
	"fmt"
)

// UserCreateRequest 创建用户的业务请求参数结构体
type UserCreateRequest struct {
	Role     int8   `json:"role"`     // 用户角色
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码
}

// Create 实现用户创建的业务逻辑
func (u *UserService) Create(req UserCreateRequest) (user models.UserModel, err error) {
	// 检查用户名是否已存在
	err = global.DB.Take(&user, "username = ?", req.Username).Error
	if err == nil {
		err = fmt.Errorf("%s 用户名已存在", req.Username)
		return
	}

	// 密码加密处理
	hashPwd, _ := pwd.GenerateFromPassword(req.Password)
	// 构建用户模型实例
	user = models.UserModel{
		Username: req.Username,
		Password: hashPwd,
		Role:     req.Role,
	}
	// 写入数据库创建用户
	err = global.DB.Create(&user).Error
	if err != nil {
		err = fmt.Errorf("用户创建失败 %s", err)
		return
	}
	// 记录用户创建成功日志
	u.log.Infof("%s 用户创建成功", req.Username)
	return
}
