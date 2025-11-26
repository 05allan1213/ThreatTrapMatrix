package user_api

// File: honey_server/api/user_api/create.go
// Description: 用户创建API接口

import (
	"fmt"

	"ThreatTrapMatrix/apps/honey_server/middleware"
	"ThreatTrapMatrix/apps/honey_server/service/user_service"
	"ThreatTrapMatrix/apps/honey_server/utils/response"

	"github.com/gin-gonic/gin"
)

// CreateRequest 创建用户请求参数结构体
type CreateRequest struct {
	Username string `json:"username" binding:"required" label:"用户名"` // 用户名（必填）
	Password string `json:"password" binding:"required" label:"密码"`  // 密码（必填）
	Role     int8   `json:"role" binding:"required,ne=1"`            // 用户角色（必填，不能为1）
}

// CreateView 创建用户接口处理函数
func (UserApi) CreateView(c *gin.Context) {
	// 获取绑定的创建用户请求参数
	cr := middleware.GetBind[CreateRequest](c)

	// 获取上下文日志实例
	log := middleware.GetLog(c)
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
		log.Errorf(msg)
		response.FailWithMsg(msg, c)
		return
	}
	// 返回创建成功结果（用户ID）
	response.OkWithData(user.ID, c)
}
