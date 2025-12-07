package vs_api

// File: image_server/api/vs_api/vs_create.go
// Description: 虚拟服务创建接口实现，基于Docker SDK创建容器并完成虚拟服务数据管理

import (
	"fmt"
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/service/docker_service"
	"image_server/internal/utils/response"
	"net"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// VsCreateRequest 虚拟服务创建请求参数结构体
type VsCreateRequest struct {
	ImageID uint `json:"imageID" binding:"required"` // 关联的镜像ID（必传）
}

// 虚拟服务IP地址池配置常量
const (
	maxIP = 254 // 子网中最后一段的最大可用值（如10.2.0.254）
)

// getNextAvailableIP 从配置的子网中动态获取下一个可用的IP地址
func getNextAvailableIP() (string, error) {
	// 从全局配置中解析子网信息（如10.2.0.0/24）
	ip, _, err := net.ParseCIDR(global.Config.VsNet.Net)
	if err != nil {
		return "", err
	}
	ip4 := ip.To4() // 转换为IPv4地址格式

	// 查询数据库中已分配的最大IP地址（按IP降序取第一条）
	var service models.ServiceModel
	err = global.DB.Order("ip DESC").First(&service).Error
	if err != nil {
		if err.Error() == "record not found" {
			// 无已分配记录，返回子网起始IP（最后一段设为2，如10.2.0.2）
			ip4[3] = 2
			return ip4.String(), nil
		}
		return "", fmt.Errorf("查询最大IP失败: %w", err)
	}

	// 解析数据库中已分配的服务IP
	serviceIP := net.ParseIP(service.IP)
	if serviceIP == nil {
		return "", fmt.Errorf("服务ip解析错误")
	}
	serviceIP4 := serviceIP.To4()

	// 检查是否已达到子网IP地址池上限
	if serviceIP4[3] >= maxIP {
		return "", fmt.Errorf("IP地址池已满")
	}

	// 生成新的可用IP地址（最后一段递增）
	newLastOctet := serviceIP4[3] + 1
	ip4[3] = newLastOctet
	return ip4.String(), nil
}

// VsCreateView 虚拟服务创建接口处理函数
func (VsApi) VsCreateView(c *gin.Context) {
	log := middleware.GetLog(c)

	// 获取并绑定虚拟服务创建请求参数
	cr := middleware.GetBind[VsCreateRequest](c)

	log.WithFields(map[string]interface{}{
		"image_id":     cr.ImageID,
		"request_data": cr,
	}).Info("virtual service creation request received") // 收到虚拟服务创建请求

	// 查询关联的镜像信息（校验镜像是否存在）
	var image models.ImageModel
	if err := global.DB.Take(&image, cr.ImageID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"image_id": cr.ImageID,
			"error":    err,
		}).Warn("failed to find image by ID") // 未能通过 ID 找到镜像
		response.FailWithMsg("镜像不存在", c)
		return
	}
	// 校验镜像状态是否为可用（状态2为禁用）
	if image.Status == 2 {
		log.WithFields(map[string]interface{}{
			"image_id": image.ID,
			"status":   image.Status,
		}).Warn("attempted to use unavailable image") // 尝试使用不可用的镜像
		response.FailWithMsg("镜像不可用", c)
		return
	}

	// 校验该镜像是否已创建过虚拟服务（一个镜像仅允许创建一个虚拟服务）
	var service models.ServiceModel
	if err := global.DB.Take(&service, "image_id = ?", cr.ImageID).Error; err == nil {
		log.WithFields(map[string]interface{}{
			"image_id":            cr.ImageID,
			"existing_service_id": service.ID,
		}).Warn("service already exists for this image") // 已存在该镜像的虚拟服务
		response.FailWithMsg("此镜像已运行虚拟服务", c)
		return
	}

	// 从动态IP地址池获取下一个可用IP
	ip, err := getNextAvailableIP()
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("failed to get next available IP") // 获取下一个可用IP失败
		response.FailWithMsg("IP地址池已满，无法创建新服务", c)
		return
	}
	log.WithFields(map[string]interface{}{
		"allocated_ip": ip,
	}).Info("allocated IP address for new service") // 已为新服务分配IP地址

	// 从全局配置获取Docker网络及容器名称前缀
	networkName := global.Config.VsNet.Name
	containerName := global.Config.VsNet.Prefix + image.ImageName // 容器名称（配置前缀+镜像名）

	fullImageName := fmt.Sprintf("%s:%s", image.ImageName, image.Tag)

	log.WithFields(map[string]interface{}{
		"container_name": containerName,
		"network_name":   networkName,
		"ip_address":     ip,
		"image_name":     fullImageName,
	}).Info("preparing to run container") // 准备启动容器

	// 通过Docker SDK创建并启动容器
	containerID, err := docker_service.RunContainer(containerName, networkName, ip, fullImageName)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"container_name": containerName,
			"error":          err,
		}).Error("failed to create container") // 创建容器失败
		response.FailWithMsg("创建虚拟服务失败", c)
		return
	}

	// 构建Docker命令
	command := fmt.Sprintf("docker run -d --network %s --ip %s --name %s %s",
		networkName, ip, containerName, fullImageName)
	log.WithFields(map[string]interface{}{
		"command":      command,
		"container_id": containerID,
	}).Info("container created successfully") // 容器创建成功

	// 组装虚拟服务数据模型
	serviceModel := models.ServiceModel{
		Title:         image.Title,     // 虚拟服务名称（复用镜像别名）
		ContainerName: containerName,   // Docker容器名称（配置前缀+镜像名）
		Agreement:     image.Agreement, // 通信协议（复用镜像配置）
		ImageID:       image.ID,        // 关联镜像ID
		IP:            ip,              // 动态分配的容器IP地址
		Port:          image.Port,      // 服务端口（复用镜像配置）
		Status:        1,               // 服务状态：1-运行中
		ContainerID:   containerID,     // Docker容器ID（12位短ID）
	}

	// 虚拟服务数据入库
	if err := global.DB.Create(&serviceModel).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"container_id": containerID,
			"error":        err,
		}).Error("failed to save service record to database") // 虚拟服务数据入库失败
		response.FailWithMsg("创建虚拟服务失败", c)
		return
	}
	log.WithFields(map[string]interface{}{
		"service_id":   serviceModel.ID,
		"container_id": containerID,
	}).Info("service record saved to database") // 虚拟服务数据入库成功

	// 启动一个协程，定时检查容器状态并更新数据库
	go func(model *models.ServiceModel, log *logrus.Entry) {
		delayList := []<-chan time.Time{
			time.After(5 * time.Second),
			time.After(20 * time.Second),
			time.After(1 * time.Minute),
			time.After(5 * time.Minute),
			time.After(10 * time.Minute),
			time.After(30 * time.Minute),
			time.After(1 * time.Hour),
		}
		for _, delay := range delayList {
			<-delay
			ContainerStatus(model, log)
		}
	}(&serviceModel, log)

	response.Ok(serviceModel.ID, "创建虚拟服务成功", c)
}

// ContainerStatus 单个容器状态检查与同步
func ContainerStatus(model *models.ServiceModel, log *logrus.Entry) {
	log.WithFields(map[string]interface{}{
		"container_name": model.ContainerName,
		"container_id":   model.ContainerID,
	}).Info("checking container status") // 检查容器状态

	var newModel models.ServiceModel
	// 根据容器名称前缀查询容器状态
	containers, err := docker_service.PrefixContainerStatus(model.ContainerName)

	isUpdate := false // 是否需要更新数据库标记
	state := ""       // 最新容器状态描述

	// 容器查询失败处理
	if err != nil {
		newModel.Status = 2             // 标记为异常状态
		newModel.ErrorMsg = err.Error() // 记录错误信息
		isUpdate = true
		state = err.Error()
		log.WithFields(map[string]interface{}{
			"container_name": model.ContainerName,
			"error":          err,
		}).Warn("error checking container status")
	} else if len(containers) != 1 {
		newModel.Status = 2                            // 标记为异常状态
		newModel.ErrorMsg = "container does not exist" // 记录错误信息
		isUpdate = true
		state = newModel.ErrorMsg
		log.WithFields(map[string]interface{}{
			"container_name": model.ContainerName,
			"found_count":    len(containers),
		}).Warn("container not found or multiple containers detected") // 容器不存在或找到多个匹配的容器
	} else {
		// 获取匹配的容器信息
		container := containers[0]

		if container.State == "running" && model.Status != 1 { // 场景1：数据库记录异常，但容器实际运行正常 → 同步为正常状态
			newModel.Status = 1
			newModel.ErrorMsg = ""
			isUpdate = true
			state = container.State
		} else if container.State != "running" && model.Status == 1 { // 场景2：数据库记录正常，但容器实际异常 → 同步为异常状态
			newModel.Status = 2
			newModel.ErrorMsg = fmt.Sprintf("%s(%s)", container.State, container.Status)
			isUpdate = true
			state = container.State
		}
		log.WithFields(map[string]interface{}{
			"container_name": model.ContainerName,
			"state":          container.State,
			"status":         container.Status,
		}).Info("container status checked") // 容器状态检查完毕
	}

	// 存在状态差异时更新数据库
	if isUpdate {
		oldState := model.State() // Assuming State() method returns string representation
		log.WithFields(map[string]interface{}{
			"container_name": model.ContainerName,
			"old_state":      oldState,
			"new_state":      state,
		}).Info("container status updated") // 容器状态更新

		if err := global.DB.Model(model).Updates(map[string]interface{}{
			"status":    newModel.Status,
			"error_msg": newModel.ErrorMsg,
		}).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"container_name": model.ContainerName,
				"error":          err,
			}).Error("failed to update container status in database") // 数据库更新容器状态失败
		}
	}
}
