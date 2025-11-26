package models

// LogModel 系统日志模型
type LogModel struct {
	Model
	Type        int8   `json:"type"`        // 日志类型
	IP          string `json:"ip"`          // ip（登录日志）
	Addr        string `json:"addr"`        // 地址
	UserID      uint   `json:"userID"`      // 用户id
	Username    string `json:"username"`    // 用户名
	Pwd         string `json:"pwd"`         // 密码（输入错误）
	LoginStatus bool   `json:"loginStatus"` // 登录状态
	Title       string `json:"title"`       // 日志别名（操作日志）
	Level       int8   `json:"level"`       // 级别（操作日志）
	Content     string `json:"content"`     // 操作详情（操作日志）
	ServiceName string `json:"serviceName"` // 服务名称（运行日志）
}
