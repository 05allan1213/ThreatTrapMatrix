package mirror_cloud_api

// File: image_server/api/mirror_cloud_api/image_create.go
// Description: 镜像创建API接口

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/middleware"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/utils/path"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ImageCreateRequest 镜像创建接口请求参数结构体
type ImageCreateRequest struct {
	ImageID   string `json:"imageID" binding:"required"`   // 镜像ID
	ImageName string `json:"imageName" binding:"required"` // 镜像名称
	ImageTag  string `json:"imageTag" binding:"required"`  // 镜像标签
	ImagePath string `json:"imagePath" binding:"required"` // 镜像临时上传路径
	Title     string `json:"title" binding:"required"`     // 镜像别名
	Port      int    `json:"port" binding:"required"`      // 镜像运行端口
	Agreement int8   `json:"agreement" binding:"required"` // 镜像通信协议
}

// ImageCreateView 镜像创建接口处理函数
func (MirrorCloudApi) ImageCreateView(c *gin.Context) {
	// 获取并绑定请求参数
	cr := middleware.GetBind[ImageCreateRequest](c)

	// 1. 校验镜像临时文件是否存在
	if _, err := os.Stat(cr.ImagePath); errors.Is(err, os.ErrNotExist) {
		response.FailWithMsg("镜像文件不存在", c)
		return
	}

	// 2. 校验镜像别名是否重复（保证title唯一性）
	var titleExists models.ImageModel
	if err := global.DB.Take(&titleExists, "title = ?", cr.Title).Error; err == nil {
		response.FailWithMsg("镜像别名不能重复", c)
		return
	}

	// 3. 校验镜像名称+标签组合是否重复（保证镜像标识唯一性）
	var nameTagExists models.ImageModel
	if err := global.DB.Take(&nameTagExists, "image_name = ? AND tag = ?", cr.ImageName, cr.ImageTag).Error; err == nil {
		response.FailWithMsg("镜像名称和标签组合不能重复", c)
		return
	}

	// 4. 执行docker load命令将镜像文件导入Docker引擎
	cmd := exec.Command("docker", "load", "-i", cr.ImagePath)
	// 设置命令执行目录为项目根路径
	cmd.Dir = path.GetRootPath()
	output, err := cmd.CombinedOutput()
	if err != nil {
		response.FailWithMsg(fmt.Sprintf("镜像导入失败: %s, 输出: %s", err.Error(), string(output)), c)
		return
	}
	fmt.Println(string(output))

	// 5. 将临时镜像文件迁移至正式存储目录
	finalDir := "uploads/images/"

	// 确保正式存储目录存在（不存在则创建）
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		response.FailWithMsg(fmt.Sprintf("创建目标目录失败: %s", err.Error()), c)
		return
	}

	// 提取临时文件的文件名
	_, fileName := filepath.Split(cr.ImagePath)
	finalPath := filepath.Join(finalDir, fileName)

	// 移动临时文件到正式目录
	if err := os.Rename(cr.ImagePath, finalPath); err != nil {
		// 移动失败时记录错误日志
		logrus.Errorf("文件移动失败 %s", err)
		response.FailWithMsg("文件移动失败", c)
		return
	}

	// 6. 组装镜像数据并写入数据库
	imageModel := models.ImageModel{
		DockerImageID: cr.ImageID,
		ImageName:     cr.ImageName,
		Tag:           cr.ImageTag,
		ImagePath:     finalPath,
		Title:         cr.Title,
		Port:          cr.Port,
		Agreement:     cr.Agreement,
		Status:        1, // 1表示镜像状态为可用
	}

	if err := global.DB.Create(&imageModel).Error; err != nil {
		response.FailWithMsg(fmt.Sprintf("数据库插入失败: %s", err.Error()), c)
		return
	}

	// 返回创建成功响应，包含镜像ID及提示信息
	response.Ok(imageModel.ID, "镜像创建成功", c)
}
