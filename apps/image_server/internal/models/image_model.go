package models

import (
	"fmt"
	"image_server/internal/utils/cmd"
	"os"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ImageModel 镜像模型
type ImageModel struct {
	Model
	ImageName     string         `gorm:"size:64" json:"imageName"`     // 镜像名称
	Title         string         `gorm:"size:64" json:"title"`         // 镜像对外别名
	Port          int            `json:"port"`                         // 端口号
	DockerImageID string         `gorm:"size:32" json:"dockerImageID"` // docker镜像id
	ServiceList   []ServiceModel `gorm:"foreignKey:ImageID" json:"-"`  // 关联的虚拟服务列表
	Tag           string         `gorm:"size:32" json:"tag"`           // 镜像标签
	Agreement     int8           `json:"agreement"`                    // 镜像通信协议
	ImagePath     string         `gorm:"size:256" json:"-"`            // 镜像文件
	Status        int8           `json:"status"`                       // 镜像状态 1 成功
	Logo          string         `gorm:"size:256" json:"logo"`         // 镜像logo
	Desc          string         `gorm:"size:512" json:"desc"`         // 镜像描述
}

func (i *ImageModel) BeforeDelete(tx *gorm.DB) error {
	// 删除docker镜像
	command := fmt.Sprintf("docker rmi %s", i.DockerImageID)
	err := cmd.Cmd(command)
	if err != nil {
		// 即使删除镜像失败也继续执行，因为可能镜像已经被删除或其他原因
		logrus.Warnf("删除docker镜像失败 %s: %v", i.DockerImageID, err)
	}
	
	// 删除镜像文件
	logrus.Infof("删除镜像文件 %s", i.ImagePath)
	err = os.Remove(i.ImagePath)
	if err != nil {
		// 文件可能已经不存在，这不应该阻止删除操作
		logrus.Warnf("删除镜像文件失败 %s: %v", i.ImagePath, err)
	}
	
	return nil
}
