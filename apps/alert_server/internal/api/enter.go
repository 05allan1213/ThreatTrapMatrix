package api

import (
	"alert_server/internal/api/white_ip_api"
)

// File:alert_server/api/user_api/enter.go
// Description: 系统Api入口

// Api 全局Api定义
type Api struct {
	WhiteIPApi white_ip_api.WhiteIPApi
}

var App = Api{}
