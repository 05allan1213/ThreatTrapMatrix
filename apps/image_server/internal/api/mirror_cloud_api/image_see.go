package mirror_cloud_api

// File: image_server/api/mirror_cloud_api/image_see.go
// Description: 镜像文件查看API接口实现，处理镜像文件上传、验证、解析及临时文件管理

import (
	"ThreatTrapMatrix/apps/image_server/internal/utils/docker"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"ThreatTrapMatrix/apps/image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// ImageSeeResponse 镜像查看接口响应结构体
type ImageSeeResponse struct {
	ImageID   string `json:"imageID"`   // 镜像ID
	ImageName string `json:"imageName"` // 镜像名称
	ImageTag  string `json:"imageTag"`  // 镜像标签
	ImagePath string `json:"imagePath"` // 镜像临时上传路径
}

const (
	maxFileSize  = 2 << 30                // 镜像文件大小上限：2GB
	tempImageDir = "uploads/images_temp/" // 镜像临时存储目录
)

// ImageSeeView 镜像查看接口处理函数
func (MirrorCloudApi) ImageSeeView(c *gin.Context) {
	// 从表单获取上传的镜像文件
	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithMsg("请选择镜像文件", c)
		return
	}

	// 校验文件大小是否超出限制
	if file.Size > maxFileSize {
		response.FailWithMsg("镜像文件大小不能超过2GB", c)
		return
	}

	// 校验文件格式是否为支持的镜像包格式
	ext := filepath.Ext(file.Filename)
	if ext != ".tar" && ext != ".gz" {
		response.FailWithMsg("只支持.tar和.tar.gz格式的镜像文件", c)
		return
	}

	// 创建临时目录（若不存在）
	if err := os.MkdirAll(tempImageDir, 0755); err != nil {
		response.FailWithMsg(fmt.Sprintf("创建临时目录失败: %v", err), c)
		return
	}

	// 拼接临时文件路径并保存上传的镜像文件
	tempFilePath := filepath.Join(tempImageDir, file.Filename)
	if err := c.SaveUploadedFile(file, tempFilePath); err != nil {
		response.FailWithMsg(fmt.Sprintf("保存镜像文件失败: %v", err), c)
		return
	}

	// 解析镜像文件的元数据（ID、名称、标签）
	imageID, imageName, imageTag, err := docker.ParseImageMetadata(tempFilePath)
	if err != nil {
		// 解析失败时清理已保存的临时文件
		os.Remove(tempFilePath)
		response.FailWithMsg(fmt.Sprintf("解析镜像元数据失败: %v", err), c)
		return
	}

	// 异步执行临时文件清理（延迟5分钟）
	go func() {
		time.Sleep(5 * time.Minute)
		err = os.Remove(tempFilePath)
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			logrus.Errorf("镜像删除失败 %s", err)
		} else {
			logrus.Infof("删除镜像文件 %s", tempFilePath)
		}
	}()

	// 组装接口响应数据
	data := ImageSeeResponse{
		ImageID:   imageID,
		ImageName: imageName,
		ImageTag:  imageTag,
		ImagePath: tempFilePath,
	}

	// 返回成功响应及镜像信息
	response.OkWithData(data, c)
}
