package api

// File:image_server/api/enter.go
// Description: 系统Api入口

import (
	"ThreatTrapMatrix/apps/image_server/internal/api/mirror_cloud_api"
	"ThreatTrapMatrix/apps/image_server/internal/api/vs_api"
	"ThreatTrapMatrix/apps/image_server/internal/api/vs_net_api"
)

// Api 全局Api定义
type Api struct {
	MirrorCloudApi mirror_cloud_api.MirrorCloudApi
	VsApi          vs_api.VsApi
	VsNetApi       vs_net_api.VsNetApi
}

var App = Api{}
