package models

import (
	"ThreatTrapMatrix/apps/image_server/internal/utils/cmd"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ServiceModel 虚拟服务模型
type ServiceModel struct {
	Model
	Title         string     `json:"title"`                       // 虚拟服务名称
	Agreement     int8       `json:"agreement"`                   // 协议
	ImageID       uint       `json:"imageID"`                     // 使用的镜像id
	ImageModel    ImageModel `gorm:"foreignKey:ImageID" json:"-"` // 使用的镜像
	IP            string     `json:"ip"`                          // 虚拟ip
	Port          int        `json:"port"`                        // 端口号
	Status        int8       `json:"status"`                      // 运行状态
	ErrorMsg      string     `json:"errorMsg"`                    // 错误信息
	HoneyIPCount  int        `json:"honeyIPCount"`                // 关联诱捕ip数量
	ContainerID   string     `json:"containerID"`                 // 容器id
	ContainerName string     `json:"containerName"`               // 容器名称
}

// State 获取虚拟服务状态
func (s *ServiceModel) State() string {
	switch s.Status {
	case 1:
		return "running"
	}
	return "error"
}

func (s *ServiceModel) BeforeDelete(tx *gorm.DB) error {
	// 判断是否有关联的端口转发
	var count int64
	tx.Model(HoneyPortModel{}).Where("service_id = ?", s.ID).Count(&count)
	if count > 0 {
		return errors.New("存在端口转发，不能删除虚拟服务")
	}

	command := fmt.Sprintf("docker rm -f %s", s.ContainerName)
	err := cmd.Cmd(command)
	if err != nil {
		logrus.Errorf("删除容器失败 %s", err)
		return err
	}
	return nil
}
