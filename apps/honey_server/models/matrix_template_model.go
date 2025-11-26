package models

// MatrixTemplateModel 矩阵模板模型
type MatrixTemplateModel struct {
	Model
	Title            string           `gorm:"size:32" json:"title"`                    // 矩阵模板名称
	HostTemplateList HostTemplateList `gorm:"serializer:json" json:"hostTemplateList"` // 主机模板列表
}

// HostTemplateList 主机模板列表
type HostTemplateList []HostTemplateInfo

// HostTemplateInfo 主机模板信息详情模型
type HostTemplateInfo struct {
	HostTemplateID uint `json:"hostTemplateID"` // 主机模板id
	Weight         int  `json:"weight"`         // 主机模板所占权重
}
