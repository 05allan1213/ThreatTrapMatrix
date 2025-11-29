package log_service

// File: honey_server/service/log_service/login_log.go
// Description: 日志服务模块，负责用户登录日志的记录与管理，包括成功/失败登录日志的存储

import (
	"honey_server/internal/core"
	"honey_server/internal/global"
	"honey_server/internal/models"

	"github.com/gin-gonic/gin"
)

// LoginLogService 登录日志服务结构体
type LoginLogService struct {
	IP   string // 客户端IP地址
	Addr string // 客户端地理位置
}

// NewLoginLog 创建LoginLogService实例的构造函数
func NewLoginLog(c *gin.Context) *LoginLogService {
	return &LoginLogService{
		IP:   c.ClientIP(),                 // 从上下文获取客户端IP
		Addr: core.GetIpAddr(c.ClientIP()), // 地理位置信息
	}
}

// SuccessLog 记录登录成功日志
func (l LoginLogService) SuccessLog(userID uint, username string) {
	l.save(userID, username, "", "登录成功", true)
}

// FailLog 记录登录失败日志
func (l LoginLogService) FailLog(username string, password string, title string) {
	l.save(0, username, password, title, false)
}

// save 内部日志存储方法，统一处理登录日志的持久化
func (l LoginLogService) save(userID uint, username string, password string, title string, loginStatus bool) {
	// 创建登录日志记录并写入数据库
	global.DB.Create(&models.LogModel{
		Type:        1,           // 日志类型：1-登录日志
		IP:          l.IP,        // 客户端IP
		Addr:        l.Addr,      // 客户端地址
		UserID:      userID,      // 用户ID
		Username:    username,    // 用户名
		Pwd:         password,    // 密码
		LoginStatus: loginStatus, // 登录状态
		Title:       title,       // 日志描述
	})
}
