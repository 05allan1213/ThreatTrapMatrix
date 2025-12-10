package models

// WhiteIPModel 白名单IP模型
type WhiteIPModel struct {
	Model
	IP     string `gorm:"size:32" json:"ip"`     // 白名单IP
	Notice string `gorm:"size:64" json:"notice"` // 备注信息
}
