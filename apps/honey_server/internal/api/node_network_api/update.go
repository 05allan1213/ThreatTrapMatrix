package node_network_api

// File: honey_server/api/node_network_api/update.go
// Description: 节点网卡信息更新API接口

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"
	"net"

	"github.com/gin-gonic/gin"
)

// UpdateRequest 网卡信息更新请求参数结构体
type UpdateRequest struct {
	ID      uint   `json:"id" binding:"required"` // 网卡ID(必填)
	Gateway string `json:"gateway"`               // 网关(选填)
}

// UpdateView 处理节点网卡信息更新请求，主要校验并更新网关配置
func (NodeNetworkApi) UpdateView(c *gin.Context) {
	log := middleware.GetLog(c)
	// 绑定并验证请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	log.WithFields(map[string]interface{}{
		"network_id":  cr.ID,
		"new_gateway": cr.Gateway,
	}).Info("node network interface update request received") // 收到节点网卡信息更新请求

	var model models.NodeNetworkModel
	// 查询指定ID的网卡记录
	if err := global.DB.Take(&model, cr.ID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"network_id": cr.ID,
			"error":      err,
		}).Warn("node network interface not found") // 未找到节点网卡
		response.FailWithMsg("节点网卡不存在", c)
		return
	}

	// 网关参数非空时执行合法性校验
	if cr.Gateway != "" {
		// 验证网关IP格式有效性
		gateway := net.ParseIP(cr.Gateway)
		if gateway == nil {
			log.WithFields(map[string]interface{}{
				"network_id":      cr.ID,
				"invalid_gateway": cr.Gateway,
			}).Warn("invalid gateway IP format") // 网关ip格式错误
			response.FailWithMsg("网关ip格式错误", c)
			return
		}

		// 验证网关必须为IPv4地址
		ip4 := gateway.To4()
		if ip4 == nil {
			log.WithFields(map[string]interface{}{
				"network_id": cr.ID,
				"gateway":    cr.Gateway,
			}).Warn("gateway is not an IPv4 address") // 网关ip不是ipv4地址
			response.FailWithMsg("网关ip只支持ipv4", c)
			return
		}

		// 验证网关不能与探针自身IP相同
		if cr.Gateway == model.IP {
			log.WithFields(map[string]interface{}{
				"network_id": cr.ID,
				"gateway":    cr.Gateway,
				"probe_ip":   model.IP,
			}).Warn("gateway cannot be the same as probe IP") // 网关不能与探针IP相同
			response.FailWithMsg("网关ip不能是探针ip", c)
			return
		}

		// 验证网关属于当前网卡所在子网
		_, _net, err := net.ParseCIDR(fmt.Sprintf("%s/%d", model.IP, model.Mask))
		if err != nil {
			log.WithFields(map[string]interface{}{
				"network_id": cr.ID,
				"ip":         model.IP,
				"mask":       model.Mask,
				"error":      err,
			}).Error("failed to parse network CIDR") // 解析网络 CIDR 失败
			response.FailWithMsg("解析子网信息失败", c)
			return
		}

		if !_net.Contains(gateway) {
			log.WithFields(map[string]interface{}{
				"network_id":   cr.ID,
				"gateway":      cr.Gateway,
				"network_cidr": fmt.Sprintf("%s/%d", model.IP, model.Mask),
			}).Warn("gateway is not in the current subnet") // 网关ip不属于当前子网
			response.FailWithMsg("网关ip不属于当前子网", c)
			return
		}
	}

	// 更新网卡网关配置
	if err := global.DB.Model(&model).Update("gateway", cr.Gateway).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"network_id": cr.ID,
			"gateway":    cr.Gateway,
			"error":      err,
		}).Error("failed to update node network interface") // 更新节点网卡失败
		response.FailWithMsg("节点网卡修改失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"network_id":      cr.ID,
		"updated_gateway": cr.Gateway,
	}).Info("node network interface updated successfully") // 节点网卡修改成功
	response.OkWithMsg("节点网卡修改成功", c)
}
