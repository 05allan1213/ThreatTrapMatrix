package user_api

// File: honey_server/api/user_api/login.go
// Description: 用户登录接口

import (
	"fmt"

	"ThreatTrapMatrix/apps/honey_server/utils/response"

	"ThreatTrapMatrix/apps/honey_server/middleware"

	"github.com/gin-gonic/gin"
)

// LoginRequest 登录请求结构体
type LoginRequest struct {
	Username string `json:"username" binding:"required" label:"用户名"`
	Password string `json:"password" binding:"required" label:" 密码"`
}

func (UserApi) LoginView(c *gin.Context) {
	cr := middleware.GetBind[LoginRequest](c)
	log := middleware.GetLog(c)
	log.Infof("这是请求的内容 %v", cr)
	log.Infof("ip %s", c.ClientIP())

	fmt.Println(cr)
	response.OkWithMsg("登录成功", c)
}
