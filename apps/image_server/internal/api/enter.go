package api

// File:honey_server/api/user_api/enter.go
// Description: 系统Api入口

import (
	"ThreatTrapMatrix/apps/image_server/internal/api/mirror_cloud_api"
)

// Api 全局Api定义
type Api struct {
	MirrorCloudApi mirror_cloud_api.MirrorCloudApi
}

var App = Api{}
