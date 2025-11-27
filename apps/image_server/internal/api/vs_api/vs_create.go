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
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// VsCreateRequest 虚拟服务创建请求参数结构体
type VsCreateRequest struct {
	ImageID uint `json:"imageID" binding:"required"` // 关联的镜像ID（必传）
}

// 基础IP地址配置常量（用于虚拟服务IP地址池管理）
const (
	baseIP           = "10.2.0.0" // 子网基础IP（10.2.0.0/24网段）
	netmask          = 24         // 子网掩码（255.255.255.0）
	startIP          = 2          // IP地址池起始分配位置（从10.2.0.2开始）
	maxIP            = 254        // IP地址池最大可用地址（10.2.0.254）
	responseervedIPs = 1          // 保留IP数量（排除网络地址和广播地址）
)

// getNextAvailableIP 从IP地址池获取下一个可用的IP地址
func getNextAvailableIP() (string, error) {
	// 查询数据库中已分配的最大IP地址（按IP降序取第一条）
	var service models.ServiceModel
	err := global.DB.Order("ip DESC").First(&service).Error
	if err != nil {
		if err.Error() == "record not found" {
			// 无已分配记录，返回地址池起始IP
			return "10.2.0.2", nil
		}
		return "", fmt.Errorf("查询最大IP失败: %w", err)
	}

	// 解析当前最大IP的四段式结构
	ipParts := strings.Split(service.IP, ".")
	if len(ipParts) != 4 {
		return "", fmt.Errorf("无效的IP格式: %s", service.IP)
	}

	// 提取IP最后一段并转换为整数（用于递增）
	lastOctet, err := strconv.Atoi(ipParts[3])
	if err != nil {
		return "", fmt.Errorf("解析IP最后一段失败: %s", service.IP)
	}

	// 检查是否已达到地址池上限
	if lastOctet >= maxIP {
		return "", fmt.Errorf("IP地址池已满")
	}

	// 生成新的可用IP地址
	newLastOctet := lastOctet + 1
	newIP := fmt.Sprintf("10.2.0.%d", newLastOctet)
	return newIP, nil
}

// VsCreateView 虚拟服务创建接口处理函数
func (VsApi) VsCreateView(c *gin.Context) {
	// 获取并绑定虚拟服务创建请求参数
	cr := middleware.GetBind[VsCreateRequest](c)

	// 查询关联的镜像信息（校验镜像是否存在）
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

	// 从IP地址池获取下一个可用IP
	ip, err := getNextAvailableIP()
	if err != nil {
		logrus.Errorf("获取可用IP失败: %s", err)
		response.FailWithMsg("IP地址池已满，无法创建新服务", c)
		return
	}
	fmt.Println(ip)

	// Docker容器配置参数
	networkName := "honey-hy"                // 容器所属网络名称
	containerName := "hy_" + image.ImageName // 容器名称（添加业务前缀标识）

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
		IP:            ip,              // 分配的容器IP地址
		Port:          image.Port,      // 服务端口（复用镜像配置）
		Status:        1,               // 服务状态：1-运行中
		ContainerID:   containerID,     // Docker容器ID（12位短ID）
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
