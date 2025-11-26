package response

// File: honey_server/utils/response/enter.go
// Description: 统一响应格式模块，定义API接口返回数据结构及快捷响应函数

import (
	"ThreatTrapMatrix/apps/image_server/internal/utils/validate"

	"github.com/gin-gonic/gin"
)

// Response API接口统一响应结构体
type Response struct {
	Code int    `json:"code"` // 响应状态码（0表示成功，非0表示错误）
	Data any    `json:"data"` // 响应数据体
	Msg  string `json:"msg"`  // 响应消息描述
}

// response 基础响应函数，构建并返回统一格式的JSON响应
func response(code int, data any, msg string, c *gin.Context) {
	c.JSON(200, Response{
		Code: code,
		Data: data,
		Msg:  msg,
	})
}

// Ok 通用成功响应（自定义数据和消息）
func Ok(data any, msg string, c *gin.Context) {
	response(0, data, msg, c)
}

// OkWithData 成功响应（仅返回数据，默认消息）
func OkWithData(data any, c *gin.Context) {
	Ok(data, "成功", c)
}

// OkWithMsg 成功响应（仅返回消息，空数据）
func OkWithMsg(msg string, c *gin.Context) {
	Ok(gin.H{}, msg, c)
}

// OkWithList 列表数据成功响应（包含数据列表和总数）
func OkWithList(list any, count int64, c *gin.Context) {
	Ok(gin.H{"list": list, "count": count}, "成功", c)
}

// Fail 通用失败响应（自定义状态码和消息）
func Fail(code int, msg string, c *gin.Context) {
	response(code, nil, msg, c)
}

// FailWithMsg 失败响应（默认错误码，自定义消息）
func FailWithMsg(msg string, c *gin.Context) {
	response(1001, nil, msg, c)
}

// FailWithError 失败响应（默认错误码，使用错误对象消息）
func FailWithError(err error, c *gin.Context) {
	msg := validate.ValidateError(err)
	response(1001, nil, msg, c)
}
