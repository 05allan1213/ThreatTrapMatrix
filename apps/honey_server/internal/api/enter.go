package api

// File:honey_server/api/user_api/enter.go
// Description: 系统Api入口

import (
	"honey_server/internal/api/captcha_api"
	"honey_server/internal/api/honey_ip_api"
	"honey_server/internal/api/honey_port_api"
	"honey_server/internal/api/host_api"
	"honey_server/internal/api/log_api"
	"honey_server/internal/api/net_api"
	"honey_server/internal/api/node_api"
	"honey_server/internal/api/node_network_api"
	"honey_server/internal/api/user_api"
)

// Api 全局Api定义
type Api struct {
	UserApi        user_api.UserApi
	CaptchaApi     captcha_api.CaptchaApi
	LogApi         log_api.LogApi
	NodeApi        node_api.NodeApi
	NodeNetworkApi node_network_api.NodeNetworkApi
	NetApi         net_api.NetApi
	HostApi        host_api.HostApi
	HoneyIPApi     honey_ip_api.HoneyIPApi
	HoneyPortApi   honey_port_api.HoneyPortApi
}

var App = Api{}
