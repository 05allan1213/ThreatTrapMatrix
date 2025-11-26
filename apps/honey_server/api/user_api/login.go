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
	Username string `json:"username"`
	Password string `json:"password"`
}

func (UserApi) LoginView(c *gin.Context) {
	cr := middleware.GetBind[LoginRequest](c)

	fmt.Println(cr)
	response.OkWithMsg("登录成功", c)
}
