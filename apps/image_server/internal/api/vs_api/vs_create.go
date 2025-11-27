package vs_api

// File: image_server/api/vs_api/vs_create.go
// Description: 虚拟服务创建接口实现，基于Docker SDK创建容器并完成虚拟服务数据管理

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/middleware"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/service/docker_service"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// VsCreateRequest 虚拟服务创建请求参数结构体
type VsCreateRequest struct {
	ImageID uint `json:"imageID" binding:"required"` // 关联的镜像ID（必传）
}

// VsCreateView 虚拟服务创建接口处理函数
func (VsApi) VsCreateView(c *gin.Context) {
	// 获取并绑定虚拟服务创建请求参数
	cr := middleware.GetBind[VsCreateRequest](c)

	// 查询关联的镜像信息
	var image models.ImageModel
	err := global.DB.Take(&image, cr.ImageID).Error
	if err != nil {
		response.FailWithMsg("镜像不存在", c)
		return
	}
	// 校验镜像状态是否为可用（状态2为禁用）
	if image.Status == 2 {
		response.FailWithMsg("镜像不可用", c)
		return
	}

	// 校验该镜像是否已创建过虚拟服务（一个镜像仅允许创建一个虚拟服务）
	var service models.ServiceModel
	err = global.DB.Take(&service, "image_id = ?", cr.ImageID).Error
	if err == nil {
		response.FailWithMsg("此镜像已运行虚拟服务", c)
		return
	}

	// 配置Docker容器相关参数
	ip := "10.2.0.10"                        // 容器静态IP地址
	networkName := "honey-hy"                // 容器所属网络名称
	containerName := "hy_" + image.ImageName // 容器名称（前缀标识业务）

	// 通过Docker SDK创建并启动容器
	containerID, err := docker_service.RunContainer(
		containerName,
		networkName,
		ip,
		fmt.Sprintf("%s:%s", image.ImageName, image.Tag),
	)
	if err != nil {
		logrus.Errorf("创建虚拟服务失败 %s", err)
		response.FailWithMsg("创建虚拟服务失败", c)
		return
	}

	// 构建Docker命令
	command := fmt.Sprintf("docker run -d --network honey-hy --ip %s --name %s %s:%s",
		ip, image.ImageName, image.ImageName, image.Tag)
	fmt.Println(command)

	// 组装虚拟服务数据模型
	var model = models.ServiceModel{
		Title:         image.Title,     // 虚拟服务标题（复用镜像别名）
		ContainerName: containerName,   // Docker容器名称
		Agreement:     image.Agreement, // 通信协议（复用镜像配置）
		ImageID:       image.ID,        // 关联镜像ID
		IP:            ip,              // 容器IP地址
		Port:          image.Port,      // 服务端口（复用镜像配置）
		Status:        1,               // 服务状态：1-运行中
		ContainerID:   containerID,     // Docker容器ID（短ID）
	}

	// 虚拟服务数据入库
	err = global.DB.Create(&model).Error
	if err != nil {
		logrus.Errorf("创建虚拟服务失败 %s", err)
		response.FailWithMsg("创建虚拟服务失败", c)
		return
	}

	response.OkWithMsg("创建虚拟服务成功", c)
	return
}
