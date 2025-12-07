package net_api

// File: honey_server/api/net_api/update.go
// Description: 网络模块信息更新API接口

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/ip"
	"honey_server/internal/utils/response"
	"net"

	"github.com/gin-gonic/gin"
)

// UpdateRequest 网络信息更新请求参数结构体
type UpdateRequest struct {
	ID                 uint   `json:"id" binding:"required"`    // 网络ID(必需)
	Title              string `json:"title" binding:"required"` // 网络名称(必需)
	Gateway            string `json:"gateway"`                  // 网关地址(必需)
	CanUseHoneyIPRange string `json:"canUseHoneyIPRange"`       // 可用诱捕IP范围（格式如：192.168.1.1-192.168.1.100）
}

// UpdateView 处理网络信息更新请求，包含名称唯一性、网关合法性、IP范围有效性校验
func (NetApi) UpdateView(c *gin.Context) {
	log := middleware.GetLog(c)
	// 绑定并验证请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	log.WithFields(map[string]interface{}{
		"net_id":       cr.ID,
		"new_title":    cr.Title,
		"new_gateway":  cr.Gateway,
		"new_ip_range": cr.CanUseHoneyIPRange,
	}).Info("network update request received") // 收到网络信息更新请求

	var model models.NetModel
	// 查询指定ID的网络记录是否存在
	if err := global.DB.Take(&model, cr.ID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.ID,
			"error":  err,
		}).Warn("network not found") // 网络不存在
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 校验网络名称唯一性（排除自身ID）
	if cr.Title != model.Title {
		var newNet models.NetModel
		if err := global.DB.Take(&newNet, "title = ? and id <> ?", cr.Title, cr.ID).Error; err == nil {
			log.WithFields(map[string]interface{}{
				"net_id":            cr.ID,
				"conflicting_title": cr.Title,
				"conflicting_id":    newNet.ID,
			}).Warn("duplicate network title found") // 找到重复的网络名称
			response.FailWithMsg("修改的网络名称不能重复", c)
			return
		}
	}

	// 网关参数非空时执行合法性校验
	if cr.Gateway != "" {
		// 验证网关IP格式有效性
		gateway := net.ParseIP(cr.Gateway)
		if gateway == nil {
			log.WithFields(map[string]interface{}{
				"net_id":  cr.ID,
				"gateway": cr.Gateway,
			}).Warn("invalid gateway IP format") // 网关ip格式错误
			response.FailWithMsg("网关ip格式错误", c)
			return
		}

		// 验证网关必须为IPv4地址
		ip4 := gateway.To4()
		if ip4 == nil {
			log.WithFields(map[string]interface{}{
				"net_id":  cr.ID,
				"gateway": cr.Gateway,
			}).Warn("gateway is not an IPv4 address") // 网关ip不是IPv4地址
			response.FailWithMsg("网关ip只支持ipv4", c)
			return
		}

		// 验证网关不能与探针自身IP相同
		if cr.Gateway == model.IP {
			log.WithFields(map[string]interface{}{
				"net_id":   cr.ID,
				"gateway":  cr.Gateway,
				"probe_ip": model.IP,
			}).Warn("gateway cannot be the same as probe IP") // 网关不能与探针IP相同
			response.FailWithMsg("网关ip不能是探针ip", c)
			return
		}

		// 验证网关属于当前网络子网
		_, _net, err := net.ParseCIDR(model.Subnet())
		if err != nil {
			log.WithFields(map[string]interface{}{
				"net_id": cr.ID,
				"subnet": model.Subnet(),
				"error":  err,
			}).Error("failed to parse network subnet") // 解析子网信息失败
			response.FailWithMsg("解析子网信息失败", c)
			return
		}
		if !_net.Contains(gateway) {
			log.WithFields(map[string]interface{}{
				"net_id":  cr.ID,
				"gateway": cr.Gateway,
				"subnet":  model.Subnet(),
			}).Warn("gateway is not in current subnet") // 网关ip不属于当前子网
			response.FailWithMsg("网关ip不属于当前子网", c)
			return
		}
	}

	// 蜜罐IP范围参数非空时执行有效性校验
	if cr.CanUseHoneyIPRange != "" {
		// 解析IP范围字符串为具体IP列表
		ipList, err1 := ip.ParseIPRange(cr.CanUseHoneyIPRange)
		if err1 != nil {
			log.WithFields(map[string]interface{}{
				"net_id":   cr.ID,
				"ip_range": cr.CanUseHoneyIPRange,
				"error":    err1,
			}).Warn("invalid IP range format") // IP范围格式无效
			response.FailWithMsg(err1.Error(), c)
			return
		}

		// 校验每个IP是否属于当前网络子网
		for _, ipAddr := range ipList {
			if !model.InSubnet(ipAddr) {
				log.WithFields(map[string]interface{}{
					"net_id":   cr.ID,
					"ip":       ipAddr,
					"subnet":   model.Subnet(),
					"ip_range": cr.CanUseHoneyIPRange,
				}).Warn("IP address not in current subnet") // IP地址不在当前子网中
				response.FailWithMsg(fmt.Sprintf("%s不属于当前子网", ipAddr), c)
				return
			}
		}
	}

	// 更新网络信息到数据库
	updateData := map[string]any{
		"title":                  cr.Title,
		"gateway":                cr.Gateway,
		"can_use_honey_ip_range": cr.CanUseHoneyIPRange,
	}
	if err := global.DB.Model(&model).Updates(updateData).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id":      cr.ID,
			"update_data": updateData,
			"error":       err,
		}).Error("failed to update network information") // 网络信息更新失败
		response.FailWithMsg("网络信息修改失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"net_id": cr.ID,
	}).Info("network information updated successfully") // 网络信息更新成功
	response.OkWithMsg("网络信息修改成功", c)
}
