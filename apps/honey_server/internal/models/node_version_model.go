package models

import (
	"os"

	"gorm.io/gorm"
)

// NodeVersionModel 节点版本模型
type NodeVersionModel struct {
	Model
	ImageName string `json:"imageName"` // 镜像名称
	Tag       string `json:"tag"`       // 镜像标签
	FileName  string `json:"fileName"`  // 文件名
	ImageID   string `json:"imageID"`   // 镜像id
	FileSize  int64  `json:"fileSize"`  // 文件大小
	Path      string `json:"path"`      // 文件路径
}

func (node NodeVersionModel) BeforeDelete(tx *gorm.DB) error {
	if node.Path != "" {
		os.Remove(node.Path)
	}
	return nil
}
