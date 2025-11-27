package vs_net_api

// File: image_server/api/vs_net_api/enter.go
// Description: 虚拟子网配置API接口实现，提供子网信息查询及子网配置更新功能

import (
	"ThreatTrapMatrix/apps/image_server/internal/config"
	"ThreatTrapMatrix/apps/image_server/internal/core"
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/middleware"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/utils/cmd"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
	// 获取并绑定子网配置更新请求参数
	cr := middleware.GetBind[VsNetRequest](c)

	// 校验是否存在虚拟服务（存在则禁止修改子网配置）
	var serviceList []models.ServiceModel
	global.DB.Find(&serviceList)
	if len(serviceList) != 0 {
		response.FailWithMsg("存在虚拟服务，不可修改虚拟子网", c)
		return
	}

	// 删除旧的Docker虚拟网络
	command := fmt.Sprintf("docker network rm %s", global.Config.VsNet.Name)
	err := cmd.Cmd(command)
	if err != nil {
		logrus.Errorf("删除之前的虚拟网络失败 %s", err)
		response.FailWithMsg("删除之前的虚拟网络失败", c)
		return
	}

	// 创建新的Docker虚拟网络（指定驱动和子网段）
	command = fmt.Sprintf("docker network create --driver bridge --subnet %s %s",
		cr.Net, cr.Name)
	err = cmd.Cmd(command)
	if err != nil {
		logrus.Errorf("创建虚拟网络失败 %s", err)
		response.FailWithMsg("创建虚拟网络失败", c)
		return
	}

	// 更新全局配置并持久化到配置文件
	global.Config.VsNet = config.VsNet{
		Name:   cr.Name,
		Prefix: cr.Prefix,
		Net:    cr.Net,
	}
	core.SetConfig() // 保存配置到文件

	response.OkWithMsg("修改虚拟网络成功", c)
}
