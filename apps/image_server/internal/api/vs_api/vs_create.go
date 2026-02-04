package vs_api

// File: image_server/api/vs_api/vs_create.go
// Description: 虚拟服务创建接口实现，基于Docker SDK创建容器并完成虚拟服务数据管理

import (
	"errors"
	"fmt"
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/service/docker_service"
	"image_server/internal/utils/response"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
func getNextAvailableIP(db *gorm.DB) (string, error) {
	ip, _, err := net.ParseCIDR(global.Config.VsNet.Net)
	if err != nil {
		return "", err
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return "", fmt.Errorf("子网解析失败")
	}

	// 读取已分配IP列表，避免并发下只看最大值导致重复
	var ipList []string
	if err := db.Model(&models.ServiceModel{}).Pluck("ip", &ipList).Error; err != nil {
		return "", fmt.Errorf("查询已分配IP失败: %w", err)
	}

	// 只统计当前子网内的IP，计算最后一段最大值
	maxLastOctet := byte(1)
	for _, item := range ipList {
		serviceIP := net.ParseIP(item)
		if serviceIP == nil {
			continue
		}
		serviceIP4 := serviceIP.To4()
		if serviceIP4 == nil {
			continue
		}
		if serviceIP4[0] != ip4[0] || serviceIP4[1] != ip4[1] || serviceIP4[2] != ip4[2] {
			continue
		}
		if serviceIP4[3] > maxLastOctet {
			maxLastOctet = serviceIP4[3]
		}
	}

	if maxLastOctet >= maxIP {
		return "", fmt.Errorf("IP地址池已满")
	}
	ip4[3] = maxLastOctet + 1
	if ip4[3] < 2 {
		ip4[3] = 2
	}
	return ip4.String(), nil
}

// reserveServiceRecord 预占IP并落库，依赖唯一索引处理并发冲突
func reserveServiceRecord(image models.ImageModel, containerName string) (*models.ServiceModel, error) {
	for i := 0; i < 3; i++ {
		// 先计算候选IP
		ip, err := getNextAvailableIP(global.DB)
		if err != nil {
			return nil, err
		}
		model := models.ServiceModel{
			Title:         image.Title,
			ContainerName: containerName,
			Agreement:     image.Agreement,
			ImageID:       image.ID,
			IP:            ip,
			Port:          image.Port,
			Status:        0,
		}
		// 依赖唯一索引，重复则重试
		if err := global.DB.Create(&model).Error; err != nil {
			if isDuplicateKeyError(err) {
				continue
			}
			return nil, err
		}
		return &model, nil
	}
	return nil, fmt.Errorf("IP地址分配冲突，请重试")
}

// isDuplicateKeyError 判断是否为数据库唯一索引冲突错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	return strings.Contains(err.Error(), "Duplicate entry")
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

	// 从全局配置获取Docker网络及容器名称前缀
	networkName := global.Config.VsNet.Name
	containerName := global.Config.VsNet.Prefix + image.ImageName // 容器名称（配置前缀+镜像名）
	fullImageName := fmt.Sprintf("%s:%s", image.ImageName, image.Tag)

	// 先预留IP并落库，避免并发分配冲突
	serviceModel, err := reserveServiceRecord(image, containerName)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("failed to reserve service record")
		msg := "IP地址分配失败"
		if strings.Contains(err.Error(), "IP地址池已满") {
			msg = "IP地址池已满，无法创建新服务"
		}
		response.FailWithMsg(msg, c)
		return
	}
	ip := serviceModel.IP
	log.WithFields(map[string]interface{}{
		"allocated_ip": ip,
		"service_id":   serviceModel.ID,
	}).Info("reserved IP address for new service")

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
		// 容器创建失败，回滚服务记录
		if rollbackErr := global.DB.Delete(serviceModel).Error; rollbackErr != nil {
			log.WithFields(map[string]interface{}{
				"service_id": serviceModel.ID,
				"error":      rollbackErr,
			}).Error("failed to rollback service record")
		}
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

	// 容器创建成功，更新记录状态
	if err := global.DB.Model(serviceModel).Updates(map[string]interface{}{
		"status":       1,
		"container_id": containerID,
	}).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"service_id":   serviceModel.ID,
			"container_id": containerID,
			"error":        err,
		}).Error("failed to update service record")
		// 更新失败，清理容器避免孤儿
		if rmErr := docker_service.RemoveContainerByName(containerName); rmErr != nil {
			log.WithFields(map[string]interface{}{
				"container_name": containerName,
				"error":          rmErr,
			}).Error("failed to cleanup container")
		}
		// 再回滚服务记录
		if rollbackErr := global.DB.Delete(serviceModel).Error; rollbackErr != nil {
			log.WithFields(map[string]interface{}{
				"service_id": serviceModel.ID,
				"error":      rollbackErr,
			}).Error("failed to rollback service record")
		}
		response.FailWithMsg("创建虚拟服务失败", c)
		return
	}
	serviceModel.Status = 1
	serviceModel.ContainerID = containerID
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
	}(serviceModel, log)

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
	var state string  // 最新容器状态描述

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
