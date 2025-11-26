package models

// ImageModel 镜像模型
type ImageModel struct {
	Model
	ImageName     string `gorm:"size:64" json:"imageName"`     // 镜像名称
	Title         string `gorm:"size:64" json:"title"`         // 镜像对外别名
	Port          int    `json:"port"`                         // 端口号
	DockerImageID string `gorm:"size:32" json:"dockerImageID"` // docker镜像id
	Tag           string `gorm:"size:32" json:"tag"`           // 镜像标签
	Agreement     int8   `json:"agreement"`                    // 镜像通信协议
	ImagePath     string `gorm:"size:256" json:"imagePath"`    // 镜像文件
	Status        int8   `json:"status"`                       // 镜像状态 1 成功
	Logo          string `gorm:"size:256" json:"logo"`         // 镜像logo
	Desc          string `gorm:"size:512" json:"desc"`         // 镜像描述
}
