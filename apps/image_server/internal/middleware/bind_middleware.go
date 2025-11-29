package middleware

// File: honey_server/middleware/bind_middleware.go
// Description: 参数绑定中间件模块，提供JSON和Query参数的通用绑定及获取功能

import (
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// BindJsonMiddleware JSON参数绑定中间件，将请求体JSON数据绑定到指定类型结构体
func BindJsonMiddleware[T any](c *gin.Context) {
	var cr T
	// 将请求体JSON数据绑定到目标结构体
	err := c.ShouldBindJSON(&cr)
	if err != nil {
		// 参数绑定失败，返回错误响应并终止请求链
		response.FailWithError(err, c)
		c.Abort()
		return
	}
	// 将绑定后的结构体存入Gin上下文，供后续处理器使用
	c.Set("request", cr)
}

// BindQueryMiddleware Query参数绑定中间件，将URL查询参数绑定到指定类型结构体
func BindQueryMiddleware[T any](c *gin.Context) {
	var cr T
	// 将URL查询参数绑定到目标结构体
	err := c.ShouldBindQuery(&cr)
	if err != nil {
		// 参数绑定失败，返回错误响应并终止请求链
		response.FailWithMsg("参数绑定错误", c)
		c.Abort()
		return
	}
	// 将绑定后的结构体存入Gin上下文，供后续处理器使用
	c.Set("request", cr)
}

// BindUriMiddleware URI参数绑定中间件，将URL路径参数绑定到指定类型结构体
func BindUriMiddleware[T any](c *gin.Context) {
	var cr T
	err := c.ShouldBindUri(&cr)
	if err != nil {
		response.FailWithMsg("参数绑定错误", c)
		c.Abort()
		return
	}
	c.Set("request", cr)
}

// GetBind 从Gin上下文中获取已绑定的参数结构体
func GetBind[T any](c *gin.Context) (cr T) {
	// 从上下文获取绑定数据并类型断言为目标类型
	return c.MustGet("request").(T)
}
