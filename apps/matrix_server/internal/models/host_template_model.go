package models

// HostTemplateModel 主机模板模型
type HostTemplateModel struct {
	Model
	Title    string               `gorm:"size:64" json:"title"`            // 主机模板名称
	PortList HostTemplatePortList `gorm:"serializer:json" json:"portList"` // 开放端口组
}

// HostTemplatePortList 主机模板端口列表
type HostTemplatePortList []HostTemplatePort

// HostTemplatePort 主机模板端口模型
type HostTemplatePort struct {
	Port      int  `json:"port"`      // 端口号
	ServiceID uint `json:"serviceID"` // 关联服务ID
}
