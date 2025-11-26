package middleware

// File: honey_server/middleware/auth_middleware.go
// Description: 中间件模块，提供JWT认证和角色权限校验中间件

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/utils"
	"ThreatTrapMatrix/apps/image_server/internal/utils/jwts"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT认证中间件，验证请求头中的Token有效性
func AuthMiddleware(c *gin.Context) {
	// 检查当前请求路径是否在白名单中
	path := c.Request.URL.Path
	if utils.InList(global.Config.WhiteList, path) {
		// 在白名单中，直接放行
		c.Next()
		return
	}
	// 从请求头获取token
	token := c.GetHeader("token")
	// 解析并验证token
	claims, err := jwts.ParseToken(token)
	if err != nil {
		// 认证失败，返回错误响应并终止请求链
		response.FailWithMsg("认证失败", c)
		c.Abort()
		return
	}
	// 将解析后的claims信息存储在请求上下文中
	c.Set("claims", claims)
	// 认证通过，继续处理请求
	c.Next()
}

// GetAuth 获取当前请求的认证信息
func GetAuth(c *gin.Context) *jwts.Claims {
	return c.MustGet("claims").(*jwts.Claims)
}

// AdminMiddleware 管理员角色校验中间件，在JWT认证基础上验证用户角色
func AdminMiddleware(c *gin.Context) {
	// 从请求头获取token
	token := c.GetHeader("token")
	// 解析并验证token
	claims, err := jwts.ParseToken(token)
	if err != nil {
		// 认证失败，返回错误响应并终止请求链
		response.FailWithMsg("认证失败", c)
		c.Abort()
		return
	}
	// 校验用户角色是否为管理员（角色标识1）
	if claims.Role != 1 {
		// 角色认证失败，返回错误响应并终止请求链
		response.FailWithMsg("权限错误", c)
		c.Abort()
		return
	}
	// 角色校验通过，继续处理请求
	c.Next()
}
