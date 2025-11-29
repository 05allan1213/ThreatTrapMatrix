package vs_api

// File: image_server/api/vs_api/vs_create.go
// Description: 虚拟服务创建接口实现，基于Docker SDK创建容器并完成虚拟服务数据管理

import (
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/service/docker_service"
	"image_server/internal/utils/response"
	"fmt"
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

	// 从动态IP地址池获取下一个可用IP
	ip, err := getNextAvailableIP()
	if err != nil {
		logrus.Errorf("获取可用IP失败: %s", err)
		response.FailWithMsg("IP地址池已满，无法创建新服务", c)
		return
	}
	// 打印分配的IP
	fmt.Println(ip)

	// 从全局配置获取Docker网络及容器名称前缀
	networkName := global.Config.VsNet.Name
	containerName := global.Config.VsNet.Prefix + image.ImageName // 容器名称（配置前缀+镜像名）

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
	command := fmt.Sprintf("docker run -d --network %s --ip %s --name %s %s:%s",
		networkName, ip, containerName, image.ImageName, image.Tag)
	fmt.Println(command)

	// 组装虚拟服务数据模型
	var model = models.ServiceModel{
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
	err = global.DB.Create(&model).Error
	if err != nil {
		logrus.Errorf("创建虚拟服务失败 %s", err)
		response.FailWithMsg("创建虚拟服务失败", c)
		return
	}

	// 启动一个协程，定时检查容器状态并更新数据库
	go func(model *models.ServiceModel) {
		var delayList = []<-chan time.Time{
			time.After(5 * time.Second),
			time.After(20 * time.Second),
			time.After(1 * time.Minute),
			time.After(5 * time.Minute),
			time.After(10 * time.Minute),
			time.After(30 * time.Minute),
			time.After(1 * time.Hour),
		}
		for _, times := range delayList {
			<-times
			ContainerStatus(model)
		}
	}(&model)

	response.OkWithMsg("创建虚拟服务成功", c)
	return
}

// ContainerStatus 单个容器状态检查与同步
func ContainerStatus(model *models.ServiceModel) {
	// 记录容器状态检测日志
	logrus.Infof("检测容器状态 %s", model.ContainerName)

	var newModel models.ServiceModel
	// 根据容器名称前缀查询容器状态
	containers, err := docker_service.PrefixContainerStatus(model.ContainerName)

	var isUpdate bool // 是否需要更新数据库标记
	var state string  // 最新容器状态描述

	// 容器查询失败处理
	if err != nil {
		newModel.Status = 2             // 标记为异常状态
		newModel.ErrorMsg = err.Error() // 记录错误信息
		isUpdate = true
		state = err.Error()
	}
	// 未找到匹配容器
	if len(containers) != 1 {
		newModel.Status = 2              // 标记为异常状态
		newModel.ErrorMsg = "容器不存在" // 记录错误信息
		isUpdate = true
		state = newModel.ErrorMsg
	} else {
		// 获取匹配的容器信息
		container := containers[0]

		// 场景1：数据库记录异常，但容器实际运行正常 → 同步为正常状态
		if container.State == "running" && model.Status != 1 {
			newModel.Status = 1
			newModel.ErrorMsg = ""
			isUpdate = true
			state = container.State
		}
		// 场景2：数据库记录正常，但容器实际异常 → 同步为异常状态
		if container.State != "running" && model.Status == 1 {
			newModel.Status = 2
			newModel.ErrorMsg = fmt.Sprintf("%s(%s)", container.State, container.Status)
			isUpdate = true
			state = container.State
		}
	}

	// 存在状态差异时更新数据库
	if isUpdate {
		logrus.Infof("%s 容器存在状态修改 %s => %s", model.ContainerName, model.State(), state)
		global.DB.Model(model).Updates(map[string]any{
			"status":    newModel.Status,
			"error_msg": newModel.ErrorMsg,
		})
	}
}
