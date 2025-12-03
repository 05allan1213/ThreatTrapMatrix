package models

// ServiceModel 虚拟服务模型
type ServiceModel struct {
	Model
	Title        string     `gorm:"size:64" json:"title"`           // 虚拟服务名称
	Agreement    int8       `json:"agreement"`                      // 协议
	ImageID      uint       `json:"imageID"`                        // 使用的镜像id
	ImageModel   ImageModel `gorm:"foreignKey:ImageID" json:"-"`    // 使用的镜像
	IP           string     `gorm:"size:32;index:idx_ip" json:"ip"` // 虚拟ip
	Port         int        `json:"port"`                           // 端口号
	Status       int8       `json:"status"`                         // 运行状态
	HoneyIPCount int        `json:"honeyIPCount"`                   // 关联诱捕ip数量
	ContainerID  string     `gorm:"size:32" json:"containerID"`     // 容器id
}
