package user_api

// File: honey_server/api/user_api/logout.go
// Description: 用户注销API接口

import (
	middleware2 "ThreatTrapMatrix/apps/honey_server/internal/middleware"
	"ThreatTrapMatrix/apps/honey_server/internal/utils/response"
	"time"

	"github.com/gin-gonic/gin"
)

// UserLogoutView 用户注销接口处理函数
func (UserApi) UserLogoutView(c *gin.Context) {
	// 从请求头获取用户Token
	token := c.GetHeader("token")
	// 获取上下文日志实例
	log := middleware2.GetLog(c)
	// 获取已解析的JWT认证信息
	auth := middleware2.GetAuth(c)
	// 将Token过期时间戳转换为时间对象
	expiresAt := time.Unix(auth.ExpiresAt, 0)

	// 记录用户注销日志（包含用户ID、Token、过期时间）
	log.Infof("用户注销 %d %s %s", auth.UserID, token, expiresAt)
	// 返回注销成功响应
	response.OkWithMsg("注销成功", c)
}
