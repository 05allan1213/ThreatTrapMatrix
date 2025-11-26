package api

// File:honey_server/api/user_api/enter.go
// Description: 系统Api入口

import (
	"ThreatTrapMatrix/apps/honey_server/api/captcha_api"
	"ThreatTrapMatrix/apps/honey_server/api/log_api"
	"ThreatTrapMatrix/apps/honey_server/api/user_api"
)

// Api 全局Api定义
type Api struct {
	UserApi    user_api.UserApi
	CaptchaApi captcha_api.CaptchaApi
	LogApi     log_api.LogApi
}

var App = Api{}
