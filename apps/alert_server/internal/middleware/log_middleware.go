package middleware

// File: alert_server/middleware/log_middleware.go
// Description: 日志上下文中间件模块，为每个请求生成唯一日志ID并注入带标识的日志实例

import (
	"alert_server/internal/global"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// LogMiddleware 日志上下文中间件，为每个请求生成唯一LogID并注入带标识的日志实例
func LogMiddleware(c *gin.Context) {
	// 获取全局日志实例
	log := global.Log
	// 生成UUID作为请求唯一标识LogID
	uid := uuid.New().String()
	// 创建携带LogID的日志实例
	logger := log.WithField("logID", uid)
	// 将带标识的日志实例存入Gin上下文
	c.Set("log", logger)
}

// GetLog 从Gin上下文中获取带请求唯一标识的日志实例
func GetLog(c *gin.Context) *logrus.Entry {
	// 从上下文获取日志实例并类型断言
	return c.MustGet("log").(*logrus.Entry)
}
