package user_service

// File: honey_server/service/user_service/enter.go
// Description: 用户服务模块，封装用户相关业务逻辑的核心服务实现

import "github.com/sirupsen/logrus"

// UserService 用户服务结构体，承载用户业务逻辑处理及日志实例
type UserService struct {
	log *logrus.Entry // 日志实例，用于业务日志记录
}

// NewUserService 创建UserService实例的构造函数
func NewUserService(log *logrus.Entry) *UserService {
	return &UserService{
		log: log,
	}
}
