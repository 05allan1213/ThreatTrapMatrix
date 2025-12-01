package models

// PortModel 端口映射模型
type PortModel struct {
	Model
	LocalAddr  string `gorm:"size:64" json:"localAddr"`  // 本地监听地址（IP:Port）
	TargetAddr string `gorm:"size:64" json:"targetAddr"` // 目标服务地址（DestIP:DestPort）
}
