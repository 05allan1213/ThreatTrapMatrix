package models

// ImageModel 镜像模型
type ImageModel struct {
	Model
	ImageName string `json:"imageName"` // 镜像名称
	Title     string `json:"title"`     // 对外别名
	Port      int    `json:"port"`      // 端口号
	ImageID   string `json:"imageID"`   // 镜像id
	Tag       string `json:"tag"`       // 镜像标签
	Agreement int8   `json:"agreement"` // 协议
	ImagePath string `json:"imagePath"` // 镜像文件
	Status    int8   `json:"status"`    // 镜像状态
	Logo      string `json:"logo"`      // 镜像logo
	Desc      string `json:"desc"`      // 镜像描述
}
