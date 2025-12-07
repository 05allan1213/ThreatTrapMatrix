package vs_net_api

// File: image_server/api/vs_net_api/enter.go
// Description: 虚拟子网配置API接口实现，提供子网信息查询及子网配置更新功能

import (
	"fmt"
	"image_server/internal/config"
	"image_server/internal/core"
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/cmd"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// VsNetApi 虚拟子网配置API接口结构体
type VsNetApi struct {
}

// VsNetInfoView 获取虚拟子网配置信息接口
func (VsNetApi) VsNetInfoView(c *gin.Context) {
	response.OkWithData(global.Config.VsNet, c)
}

// VsNetRequest 虚拟子网配置更新请求参数结构体
type VsNetRequest struct {
	Name   string `json:"name" binding:"required"`   // 虚拟子网名称（Docker网络名称）
	Prefix string `json:"prefix" binding:"required"` // 容器名称前缀（用于标识业务容器）
	Net    string `json:"net" binding:"required"`    // 子网段（如10.2.0.0/24）
}

// VsNetUpdateView 更新虚拟子网配置接口
func (VsNetApi) VsNetUpdateView(c *gin.Context) {
	log := middleware.GetLog(c)

	// 获取并绑定子网配置更新请求参数
	cr := middleware.GetBind[VsNetRequest](c)

	log.WithFields(map[string]interface{}{
		"new_network_name": cr.Name,
		"new_subnet":       cr.Net,
		"new_prefix":       cr.Prefix,
	}).Info("virtual network update request received") // 收到虚拟子网配置更新请求

	// 校验是否存在虚拟服务（存在则禁止修改子网配置）
	var serviceList []models.ServiceModel
	if err := global.DB.Find(&serviceList).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("failed to query existing virtual services") // 查询虚拟服务失败
		response.FailWithMsg("查询虚拟服务失败", c)
		return
	}
	if len(serviceList) != 0 {
		log.WithFields(map[string]interface{}{
			"existing_service_count": len(serviceList),
		}).Warn("cannot update network while virtual services exist") // 存在虚拟服务，不可修改虚拟子网
		response.FailWithMsg("存在虚拟服务，不可修改虚拟子网", c)
		return
	}

	// 删除旧的Docker虚拟网络
	oldNetworkName := global.Config.VsNet.Name
	removeCmd := fmt.Sprintf("docker network rm %s", oldNetworkName)
	log.WithFields(map[string]interface{}{
		"command":      removeCmd,
		"network_name": oldNetworkName,
	}).Info("attempting to remove existing network") // 尝试删除旧的Docker虚拟网络

	if err := cmd.Cmd(removeCmd); err != nil {
		log.WithFields(map[string]interface{}{
			"error":        err,
			"command":      removeCmd,
			"network_name": oldNetworkName,
		}).Error("failed to remove existing network") // 删除旧的Docker虚拟网络失败
		response.FailWithMsg("删除之前的虚拟网络失败", c)
		return
	}

	// 创建新的Docker虚拟网络（指定驱动和子网段）
	createCmd := fmt.Sprintf("docker network create --driver bridge --subnet %s %s", cr.Net, cr.Name)
	log.WithFields(map[string]interface{}{
		"command":      createCmd,
		"network_name": cr.Name,
		"subnet":       cr.Net,
	}).Info("attempting to create new network") // 尝试创建新的Docker虚拟网络

	if err := cmd.Cmd(createCmd); err != nil {
		log.WithFields(map[string]interface{}{
			"error":        err,
			"command":      createCmd,
			"network_name": cr.Name,
			"subnet":       cr.Net,
		}).Error("failed to create new network") // 创建新的Docker虚拟网络失败
		response.FailWithMsg("创建虚拟网络失败", c)
		return
	}

	// 更新全局配置并持久化到配置文件
	global.Config.VsNet = config.VsNet{
		Name:   cr.Name,
		Prefix: cr.Prefix,
		Net:    cr.Net,
	}
	if err := core.SetConfig(); err != nil {
		log.WithFields(map[string]interface{}{
			"error":      err,
			"new_config": global.Config.VsNet,
		}).Error("failed to save updated network configuration") // 保存更新后的网络配置失败
		response.FailWithMsg("保存网络配置失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"old_network": oldNetworkName,
		"new_network": cr.Name,
		"new_subnet":  cr.Net,
	}).Info("virtual network updated successfully") // 虚拟网络修改成功

	response.OkWithMsg("修改虚拟网络成功", c)
}
