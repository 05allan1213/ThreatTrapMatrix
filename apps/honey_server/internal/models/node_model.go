package models

// NodeModel 节点模型
type NodeModel struct {
	Model
	Title        string         `gorm:"size:32" json:"title"`              // 节点名称
	Uid          string         `gorm:"size:64" json:"uid"`                // 节点uid
	IP           string         `gorm:"size:32" json:"IP"`                 // 节点ip
	Mac          string         `gorm:"size:64" json:"mac"`                // 节点mac
	Status       int8           `json:"status"`                            // 节点状态
	NetCount     int            `json:"netCount"`                          // 网络数
	HoneyIPCount int            `json:"honeyIPCount"`                      // 诱捕ip数
	Resource     NodeResource   `gorm:"serializer:json" json:"resource"`   // 节点资源占用
	SystemInfo   NodeSystemInfo `gorm:"serializer:json" json:"systemInfo"` // 节点系统信息详情
}

// NodeResource 节点资源占用模型
type NodeResource struct {
	CpuCount              int     `json:"cpuCount"`              // CPU内核数
	CpuUseRate            float64 `json:"cpuUseRate"`            // CPU使用率
	MemTotal              int64   `json:"memTotal"`              // 内存容量
	MemUseRate            float64 `json:"memUseRate"`            // 内存使用率
	DiskTotal             int64   `json:"diskTotal"`             // 磁盘容量
	DiskUseRate           float64 `json:"diskUseRate"`           // 磁盘使用率
	NodePath              string  `json:"nodePath"`              // 节点部署目录
	NodeResourceOccupancy int64   `json:"nodeResourceOccupancy"` // 节点部署目录资源占用
}

// NodeSystemInfo 节点系统信息详情模型
type NodeSystemInfo struct {
	HostName            string `json:"hostName"`            // 主机名称
	DistributionVersion string `json:"distributionVersion"` // 发行版本
	CoreVersion         string `json:"coreVersion"`         // 内核版本
	SystemType          string `json:"systemType"`          // 系统类型
	StartTime           string `json:"startTime"`           // 启动时间
	NodeVersion         string `json:"nodeVersion"`         // 节点版本
	NodeCommit          string `json:"nodeCommit"`          // 节点commit
}
