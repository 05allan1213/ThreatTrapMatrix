package models

type ServiceModel struct {
	Model
	Title         string `json:"title"`         // 镜像名称
	Agreement     int8   `json:"agreement"`     // 协议
	ImageID       uint   `json:"imageID"`       // 镜像id
	IP            string `json:"ip"`            // 容器ip
	Port          int    `json:"port"`          // 容器端口
	Status        int8   `json:"status"`        // 运行状态
	ErrorMsg      string `json:"errorMsg"`      // 错误信息
	HoneyIPCount  int    `json:"honeyIPCount"`  // 关联诱捕ip数
	ContainerID   string `json:"containerID"`   // 容器id
	ContainerName string `json:"containerName"` // 容器名称
}
