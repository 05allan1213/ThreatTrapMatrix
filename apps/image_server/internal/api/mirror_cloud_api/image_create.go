package mirror_cloud_api

// File: image_server/api/mirror_cloud_api/image_create.go
// Description: 镜像创建API接口

import (
	"errors"
	"fmt"
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/path"
	"image_server/internal/utils/response"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ImageCreateRequest 镜像创建接口请求参数结构体
type ImageCreateRequest struct {
	ImageID   string `json:"imageID" binding:"required"`              // 镜像ID（来自ImageSee接口，仅作备用标识）
	ImageName string `json:"imageName" binding:"required"`            // 镜像仓库名称
	ImageTag  string `json:"imageTag" binding:"required"`             // 镜像标签
	ImagePath string `json:"imagePath" binding:"required"`            // 镜像临时文件存储路径
	Title     string `json:"title" binding:"required"`                // 镜像展示别名
	Port      int    `json:"port" binding:"required,min=1,max=65535"` // 镜像运行端口
	Agreement int8   `json:"agreement" binding:"required,oneof=1"`    // 镜像通信协议：1-TCP协议（当前仅支持TCP）
}

// ImageCreateView 镜像创建接口处理函数
func (MirrorCloudApi) ImageCreateView(c *gin.Context) {
	// 获取并绑定请求参数
	cr := middleware.GetBind[ImageCreateRequest](c)

	log := middleware.GetLog(c)
	log.WithFields(logrus.Fields{
		"request_data": cr,
	}).Info("received image creation request") // 收到镜像创建请求

	// 1. 校验镜像临时文件是否存在
	if _, err := os.Stat(cr.ImagePath); errors.Is(err, os.ErrNotExist) {
		log.WithFields(logrus.Fields{
			"file_path": cr.ImagePath,
		}).Warn("image file does not exist") // 镜像文件不存在
		response.FailWithMsg("镜像文件不存在", c)
		return
	}

	// 2. 校验镜像别名唯一性
	var titleExists models.ImageModel
	if err := global.DB.Take(&titleExists, "title = ?", cr.Title).Error; err == nil {
		log.WithFields(logrus.Fields{
			"title": cr.Title,
		}).Warn("duplicate image title found") // 发现重复的镜像别名
		response.FailWithMsg("镜像别名不能重复", c)
		return
	}

	// 3. 校验镜像名称+标签组合唯一性
	var nameTagExists models.ImageModel
	if err := global.DB.Take(&nameTagExists, "image_name = ? AND tag = ?", cr.ImageName, cr.ImageTag).Error; err == nil {
		log.WithFields(logrus.Fields{
			"image_name": cr.ImageName,
			"tag":        cr.ImageTag,
		}).Warn("duplicate image name:tag combination found") // 发现重复的镜像名称和标签组合
		response.FailWithMsg("镜像名称和标签组合不能重复", c)
		return
	}

	// 4. 执行docker load命令导入镜像到本地Docker引擎
	loadCmd := exec.Command("docker", "load", "-i", cr.ImagePath)
	loadCmd.Dir = path.GetRootPath() // 设置命令执行工作目录为项目根路径
	log.WithFields(logrus.Fields{
		"command":     "docker load",
		"image_path":  cr.ImagePath,
		"working_dir": loadCmd.Dir,
	}).Info("executing docker load command") // 执行docker load命令
	output, err := loadCmd.CombinedOutput()
	if err != nil {
		log.WithFields(logrus.Fields{
			"error":      err,
			"output":     string(output),
			"image_path": cr.ImagePath,
		}).Error("failed to execute docker load command") // 执行docker load命令失败
		response.FailWithMsg(fmt.Sprintf("镜像导入失败: %s, 输出: %s", err, string(output)), c)
		return
	}
	log.WithFields(logrus.Fields{
		"output": string(output),
	}).Info("docker load command executed successfully") // docker load命令执行成功

	// 5. 获取Docker镜像真实短ID（12位哈希值）
	idCmd := exec.Command("docker", "images", "--quiet", cr.ImageName+":"+cr.ImageTag)
	idOutput, err := idCmd.Output()
	if err != nil {
		log.WithFields(logrus.Fields{
			"error":      err,
			"image_name": cr.ImageName,
			"image_tag":  cr.ImageTag,
		}).Error("failed to query docker image ID") // 查询镜像ID失败
		response.FailWithMsg(fmt.Sprintf("查询镜像ID失败: %s", err), c)
		return
	}

	// 处理Docker命令输出，提取短ID（去除首尾空白字符）
	actualShortID := strings.TrimSpace(string(idOutput))
	if actualShortID == "" {
		// 若Docker查询失败，使用ImageSee接口返回的ID作为备用（保证业务流程不中断）
		actualShortID = cr.ImageID
	}

	// 6. 迁移镜像临时文件到正式存储目录（持久化存储）
	finalDir := "uploads/images/"
	// 确保正式存储目录存在（不存在则创建，权限0755）
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		log.WithFields(logrus.Fields{
			"error":    err,
			"dir_path": finalDir,
		}).Error("failed to create target directory") // 创建目标目录失败
		response.FailWithMsg("创建目标目录失败: "+err.Error(), c)
		return
	}

	// 提取临时文件名称，拼接正式存储路径
	_, fileName := filepath.Split(cr.ImagePath)
	finalPath := filepath.Join(finalDir, fileName)

	// 移动临时文件到正式目录（原子操作，效率高于复制删除）
	if err := os.Rename(cr.ImagePath, finalPath); err != nil {
		log.WithFields(logrus.Fields{
			"error":       err,
			"source_path": cr.ImagePath,
			"target_path": finalPath,
		}).Error("failed to move image file") // 文件移动失败
		response.FailWithMsg("文件移动失败", c)
		return
	}

	log.WithFields(logrus.Fields{
		"source_path": cr.ImagePath,
		"target_path": finalPath,
	}).Info("image file moved to final directory") // 文件移动成功

	// 7. 组装镜像数据并写入数据库（持久化镜像配置）
	imageModel := models.ImageModel{
		DockerImageID: actualShortID, // Docker镜像真实ID
		ImageName:     cr.ImageName,
		Tag:           cr.ImageTag,
		ImagePath:     finalPath, // 镜像正式存储路径
		Title:         cr.Title,
		Port:          cr.Port,
		Agreement:     cr.Agreement,
		Status:        1, // 镜像状态：1-可用（默认创建后为可用状态）
	}

	if err := global.DB.Create(&imageModel).Error; err != nil {
		log.WithFields(logrus.Fields{
			"error":      err,
			"image_data": imageModel,
		}).Error("failed to insert image data into database") // 数据库插入失败
		response.FailWithMsg("数据库插入失败: "+err.Error(), c)
		return
	}

	log.WithFields(logrus.Fields{
		"image_id": imageModel.ID,
		"title":    imageModel.Title,
	}).Info("image data saved to database successfully") // 数据库插入成功

	// 返回创建成功响应，包含镜像业务ID及提示信息
	response.Ok(imageModel.ID, "镜像创建成功", c)
}
