package routers

// File: honey_server/routers/user_routers.go
// Description: 用户模块路由定义，注册用户相关API接口

import "github.com/gin-gonic/gin"

// UserRouters 注册用户相关路由
func UserRouters(r *gin.RouterGroup) {
	// GET /honey_server/users - 获取用户列表接口
	r.GET("users", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0})
	})
}
