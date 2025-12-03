package models

// WhiteIPModel 白名单ip模型
type WhiteIPModel struct {
	Model
	IP     string `gorm:"size:32" json:"ip"`     // 白名单ip
	Notice string `gorm:"size:64" json:"notice"` // 备注信息
}
