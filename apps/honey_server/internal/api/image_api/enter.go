package image_api

// File: honey_server/api/image_api/enter.go
// Description: 图片上传相关API接口

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"honey_server/internal/middleware"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ImageApi 图片模块API接口结构体，封装图片上传相关接口方法
type ImageApi struct{}

// ImageUploadResponse 图片上传接口响应结构体
type ImageUploadResponse struct {
	Url string `json:"url"` // 图片的访问路径（相对路径）
}

// ImageUploadView 图片上传接口处理函数
func (ImageApi) ImageUploadView(c *gin.Context) {
	// 1. 从表单中获取上传的图片文件
	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithMsg("请上传图片文件", c)
		return
	}

	// 2. 图片格式白名单校验，仅允许指定格式的图片上传
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}
	filename := file.Filename
	ext := filepath.Ext(strings.ToLower(filename)) // 统一转为小写避免格式判断错误
	if !allowedExts[ext] {
		response.FailWithMsg("不支持的图片格式，仅允许 jpg、jpeg、png、webp", c)
		return
	}

	// 3. 图片大小校验，限制单文件最大5MB
	maxSize := int64(5 * 1024 * 1024)
	if file.Size > maxSize {
		response.FailWithMsg("图片大小不能超过 5MB", c)
		return
	}

	// 4. 从登录凭证中获取用户ID
	claims := middleware.GetAuth(c)
	userID := claims.UserID
	userDir := fmt.Sprintf("%d", userID)

	// 5. 创建用户专属存储目录
	baseDir := filepath.Join("uploads", "images", userDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		response.FailWithMsg("创建存储目录失败", c)
		return
	}

	// 6. 计算上传文件的MD5哈希值，用于判断文件内容是否重复
	fileContent, err := file.Open()
	if err != nil {
		response.FailWithMsg("读取上传文件失败", c)
		return
	}
	defer fileContent.Close() // 延迟关闭文件句柄，避免资源泄漏

	hash := md5.New()
	if _, err := io.Copy(hash, fileContent); err != nil {
		response.FailWithMsg("计算文件哈希失败", c)
		return
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))

	// 7. 处理文件名，避免同目录下文件重名
	fileNameWithoutExt := strings.TrimSuffix(filepath.Base(filename), ext)
	targetPath := filepath.Join(baseDir, fileNameWithoutExt+ext)

	// 检查目标路径文件是否已存在，若存在则对比哈希值
	if _, err := os.Stat(targetPath); err == nil {
		existingFile, err := os.Open(targetPath)
		if err != nil {
			response.FailWithMsg("读取已有文件失败", c)
			return
		}
		defer existingFile.Close()

		existingHash := md5.New()
		if _, err := io.Copy(existingHash, existingFile); err != nil {
			response.FailWithMsg("计算已有文件哈希失败", c)
			return
		}
		existingFileHash := hex.EncodeToString(existingHash.Sum(nil))

		// 哈希值不同说明内容不同，添加8位UUID随机串区分文件名
		if fileHash != existingFileHash {
			randomStr := uuid.New().String()[:8]
			targetPath = filepath.Join(baseDir, fileNameWithoutExt+"_"+randomStr+ext)
		}
	}

	// 8. 将上传的文件保存到目标路径
	if err := c.SaveUploadedFile(file, targetPath); err != nil {
		response.FailWithMsg("保存图片失败", c)
		return
	}

	// 9. 拼接图片访问URL
	accessUrl := "/uploads/images/" + userDir + "/" + filepath.Base(targetPath)

	// 返回成功响应，包含图片访问路径
	response.OkWithData(ImageUploadResponse{
		Url: accessUrl,
	}, c)
}
