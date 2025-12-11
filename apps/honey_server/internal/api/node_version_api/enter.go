package node_version_api

// File: honey_server/api/node_version_api/enter.go
// Description: 节点版本管理API模块

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/docker"
	"honey_server/internal/utils/response"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NodeVersionApi 节点版本管理API结构体
type NodeVersionApi struct {
}

// 常量定义：节点镜像上传相关配置
const (
	maxFileSize  = 1 << 30               // 镜像文件最大限制：1GB
	nodeImageDir = "media/node_version/" // 节点镜像文件存储根目录，相对服务运行目录
)

// NodeVersionCreateView 节点镜像上传创建接口
func (NodeVersionApi) NodeVersionCreateView(c *gin.Context) {
	// 从表单中获取上传的镜像文件，参数名固定为"file"
	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithMsg("请上传节点镜像文件", c)
		return
	}

	// 获取请求关联的日志实例，用于上传全流程日志追踪
	log := middleware.GetLog(c)
	log.WithFields(logrus.Fields{
		"file_size": file.Size,
		"file_name": file.Filename,
	}).Info("received image file uploads request")

	// 校验文件大小：不超过1GB
	if file.Size > maxFileSize {
		response.FailWithMsg("镜像文件大小不能超过1GB", c)
		return
	}

	// 校验文件格式：仅支持.tar/.tar.gz
	ext := filepath.Ext(file.Filename)
	if ext != ".tar" && ext != ".gz" {
		response.FailWithMsg("只支持.tar和.tar.gz格式的镜像文件", c)
		return
	}

	// 创建镜像存储目录，权限0755
	if err := os.MkdirAll(nodeImageDir, 0755); err != nil {
		response.FailWithMsg(fmt.Sprintf("创建临时目录失败: %v", err), c)
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("failed to create temp directory")
		return
	}

	// 拼接镜像文件存储路径，保存上传的文件到指定目录
	nodeFilePath := filepath.Join(nodeImageDir, file.Filename)
	if err := c.SaveUploadedFile(file, nodeFilePath); err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("failed to save uploaded image file")
		response.FailWithMsg(fmt.Sprintf("保存节点镜像文件失败: %v", err), c)
		return
	}
	log.WithFields(logrus.Fields{
		"image_path": nodeFilePath,
	}).Infof("uploaded image file saved to temp path ")

	// 解析Docker镜像元数据（ImageID、镜像名、Tag），用于后续唯一性校验和存储
	imageID, imageName, imageTag, err := docker.ParseImageMetadata(nodeFilePath)
	if err != nil {
		// 解析失败时删除已保存的临时文件，避免磁盘占用
		os.Remove(nodeFilePath)
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("failed to parse image metadata")
		response.FailWithMsg(fmt.Sprintf("解析镜像元数据失败: %v", err), c)
		return
	}
	log.WithFields(logrus.Fields{
		"image_id":   imageID,
		"image_name": imageName,
		"image_tag":  imageTag,
	}).Info("image metadata parsed successfully")

	// 校验镜像Tag唯一性：同一Tag不允许重复上传
	var model models.NodeVersionModel
	err = global.DB.Take(&model, "tag = ?", imageTag).Error
	if err == nil { // 无错误表示已存在相同Tag的镜像
		response.FailWithMsg("镜像tag不能重复", c)
		return
	}

	// 构造节点版本模型，准备写入数据库
	nodeVersion := models.NodeVersionModel{
		ImageName: imageName,     // 镜像名称
		Tag:       imageTag,      // 镜像版本Tag
		FileName:  file.Filename, // 上传的文件名
		ImageID:   imageID,       // Docker镜像唯一ID
		FileSize:  file.Size,     // 文件大小（字节）
		Path:      nodeFilePath,  // 文件存储路径
	}

	// 将镜像信息写入数据库，失败则返回上传失败
	err = global.DB.Create(&nodeVersion).Error
	if err != nil {
		response.FailWithMsg("节点镜像上传失败", c)
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("节点镜像上传失败")
		return
	}
	log.WithFields(logrus.Fields{
		"data": nodeVersion,
	}).Infof("节点镜像上传成功")
	// 返回上传成功响应，数据为新创建的版本ID
	response.OkWithData(nodeVersion.ID, c)
}

// NodeVersionDownloadRequest 节点镜像下载请求结构体
type NodeVersionDownloadRequest struct {
	Version string `form:"version"` // 镜像版本Tag
	ID      string `form:"id"`      // 镜像版本ID
}

// NodeVersionDownloadView 节点镜像下载接口
func (NodeVersionApi) NodeVersionDownloadView(c *gin.Context) {
	// 绑定并解析下载请求参数（ID/Version）
	cr := middleware.GetBind[NodeVersionDownloadRequest](c)

	var model models.NodeVersionModel
	var err error

	// 优先按ID查询镜像（精准匹配）
	if cr.ID != "" {
		err = global.DB.Take(&model, cr.ID).Error
	}
	// ID为空时按Tag查询（业务维度匹配）
	if cr.Version != "" {
		err = global.DB.Take(&model, "tag = ?", cr.Version).Error
	}

	// 查询失败（无对应镜像）返回错误
	if err != nil {
		response.FailWithMsg("节点镜像不存在", c)
		return
	}
	// 未传入ID/Version参数，返回参数错误
	if model.ID == 0 {
		response.FailWithMsg("请输入version或者id参数", c)
		return
	}

	// 设置下载响应头，保证浏览器正确识别为文件下载
	c.Header("Content-Type", "application/octet-stream")                                     // 二进制流类型
	c.Header("Content-Disposition", "attachment; filename="+url.QueryEscape(model.FileName)) // 下载文件名（转义特殊字符）
	c.Header("Content-Transfer-Encoding", "binary")                                          // 二进制传输编码

	// 返回镜像文件流，Gin自动处理文件读取和响应
	c.File(model.Path)
}

// NodeVersionRemoveView 节点镜像删除接口
func (NodeVersionApi) NodeVersionRemoveView(c *gin.Context) {
	// 绑定并解析删除请求参数（仅ID）
	cr := middleware.GetBind[models.IDRequest](c)

	// 查询镜像是否存在，不存在则返回错误
	var model models.NodeVersionModel
	err := global.DB.Take(&model, cr.Id).Error
	if err != nil {
		response.FailWithMsg("节点镜像不存在", c)
		return
	}

	// 删除数据库中的镜像记录
	global.DB.Delete(&model)

	// 返回删除成功响应
	response.OkWithMsg("节点镜像删除成功", c)
}

// ListRequest 节点镜像列表查询请求结构体
type ListRequest struct {
	models.PageInfo        // 分页信息
	Tag             string `form:"tag"` // 筛选条件：镜像版本Tag
}

// NodeVersionListView 节点镜像分页列表查询接口
func (NodeVersionApi) NodeVersionListView(c *gin.Context) {
	// 绑定并解析列表查询参数（分页+Tag筛选）
	cr := middleware.GetBind[ListRequest](c)

	// 调用通用查询服务，构建查询条件：
	// 1. 筛选条件：Tag精准匹配；
	// 2. 模糊搜索：ImageName字段；
	// 3. 分页：使用前端传入的Page/PageSize；
	// 4. 排序：按创建时间降序（最新上传的在前）
	list, count, _ := common_service.QueryList(models.NodeVersionModel{
		Tag: cr.Tag,
	}, common_service.QueryListRequest{
		Likes:    []string{"image_name"}, // 模糊搜索字段：镜像名称
		PageInfo: cr.PageInfo,            // 分页参数
		Sort:     "created_at desc",      // 排序规则
	})

	// 返回分页列表响应（列表数据+总条数）
	response.OkWithList(list, count, c)
}

// OptionsResponse 节点镜像下拉选项响应结构体
type OptionsResponse struct {
	Label string `json:"label"` // 显示文本：镜像名:Tag
	Value string `json:"value"` // 提交值：镜像Tag
}

// NodeVersionOptionsView 节点镜像下拉选项查询接口
func (NodeVersionApi) NodeVersionOptionsView(c *gin.Context) {
	// 查询所有节点镜像记录
	var nodeList []models.NodeVersionModel
	global.DB.Find(&nodeList)

	// 转换为下拉选项格式
	var list = make([]OptionsResponse, 0)
	for _, model := range nodeList {
		list = append(list, OptionsResponse{
			Value: model.Tag,                                        // 提交值为Tag
			Label: fmt.Sprintf("%s:%s", model.ImageName, model.Tag), // 显示文本为"镜像名:Tag"
		})
	}

	// 返回下拉选项列表数据
	response.OkWithData(list, c)
}
