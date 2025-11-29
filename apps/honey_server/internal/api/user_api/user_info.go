package user_api

// File: honey_server/api/user_api/user_info.go
// Description: 用户信息详情API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// UserInfoResponse 用户信息响应结构体
type UserInfoResponse struct {
	UserID        uint   `json:"userID"`        // 用户ID
	Username      string `json:"username"`      // 用户名
	Role          int8   `json:"role"`          // 角色类型：1 管理员 2 普通用户
	LastLoginDate string `json:"lastLoginDate"` // 最后登录时间
}

// UserInfoView 查询当前登录用户信息接口
func (UserApi) UserInfoView(c *gin.Context) {
	// 从上下文获取已认证的用户信息
	auth := middleware.GetAuth(c)
	var user models.UserModel
	// 从数据库查询用户信息
	err := global.DB.Take(&user, auth.UserID).Error
	if err != nil {
		// 查询失败返回错误信息
		response.FailWithMsg("用户不存在", c)
		return
	}

	// 组装用户信息响应数据
	data := UserInfoResponse{
		UserID:        user.ID,
		Username:      user.Username,
		Role:          user.Role,
		LastLoginDate: user.LastLoginDate,
	}
	// 返回成功响应及用户信息数据
	response.OkWithData(data, c)
}
