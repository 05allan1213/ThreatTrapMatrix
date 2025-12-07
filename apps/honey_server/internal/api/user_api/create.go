package user_api

// File: honey_server/api/user_api/create.go
// Description: 用户创建API接口

import (
	"fmt"
	"honey_server/internal/middleware"
	"honey_server/internal/service/user_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// CreateRequest 创建用户请求参数结构体
type CreateRequest struct {
	Username string `json:"username" binding:"required" label:"用户名"` // 用户名（必填）
	Password string `json:"password" binding:"required" label:"密码"`   // 密码（必填）
	Role     int8   `json:"role" binding:"required,ne=1"`               // 用户角色（必填，不能为1）
}

// CreateView 创建用户接口处理函数
func (UserApi) CreateView(c *gin.Context) {
	// 获取绑定的创建用户请求参数
	cr := middleware.GetBind[CreateRequest](c)

	// 获取上下文日志实例
	log := middleware.GetLog(c)
	log.WithFields(map[string]interface{}{
		"username": cr.Username,
		"role":     cr.Role,
	}).Info("user creation request received") // 收到用户创建请求
	// 初始化用户服务
	us := user_service.NewUserService(log)
	// 调用服务层创建用户方法
	user, err := us.Create(user_service.UserCreateRequest{
		Username: cr.Username,
		Password: cr.Password,
		Role:     cr.Role,
	})
	if err != nil {
		msg := fmt.Sprintf("创建用户失败 %s", err)
		log.WithFields(map[string]interface{}{
			"username": cr.Username,
			"error":    err,
		}).Error("failed to create user") // 创建用户失败
		response.FailWithMsg(msg, c)
		return
	}

	log.WithFields(map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
	}).Info("user created successfully") // 用户创建成功
	// 返回创建成功结果（用户ID）
	response.OkWithData(user.ID, c)
}
