package api

// File:honey_server/api/user_api/enter.go
// Description: 系统Api入口

import "ThreatTrapMatrix/apps/honey_server/api/user_api"

// Api 全局Api定义
type Api struct {
	UserApi user_api.UserApi
}

var App = Api{}
