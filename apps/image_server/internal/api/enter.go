package api

// File:honey_server/api/user_api/enter.go
// Description: 系统Api入口

import (
	"ThreatTrapMatrix/apps/image_server/internal/api/mirror_cloud_api"
	"ThreatTrapMatrix/apps/image_server/internal/api/vs_api"
)

// Api 全局Api定义
type Api struct {
	MirrorCloudApi mirror_cloud_api.MirrorCloudApi
	VsApi          vs_api.VsApi
}

var App = Api{}
