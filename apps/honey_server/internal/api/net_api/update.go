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
	// 绑定并验证请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	var model models.NetModel
	// 查询指定ID的网络记录是否存在
	err := global.DB.Take(&model, cr.ID).Error
	if err != nil {
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 校验网络名称唯一性（排除自身ID）
	if cr.Title != model.Title {
		var newNet models.NetModel
		err = global.DB.Take(&newNet, "title = ? and id <> ?", cr.Title, cr.ID).Error
		if err == nil {
			response.FailWithMsg("修改的网络名称不能重复", c)
			return
		}
	}

	// 网关参数非空时执行合法性校验
	if cr.Gateway != "" {
		// 验证网关IP格式有效性
		gateway := net.ParseIP(cr.Gateway)
		if gateway == nil {
			response.FailWithMsg("网关ip格式错误", c)
			return
		}

		// 验证网关必须为IPv4地址
		ip4 := gateway.To4()
		if ip4 == nil {
			response.FailWithMsg("网关ip只支持ipv4", c)
			return
		}

		// 验证网关不能与探针自身IP相同
		if cr.Gateway == model.IP {
			response.FailWithMsg("网关ip不能是探针ip", c)
			return
		}

		// 验证网关属于当前网络子网
		_, _net, _ := net.ParseCIDR(model.Subnet())
		if !_net.Contains(gateway) {
			response.FailWithMsg("网关ip不属于当前子网", c)
			return
		}
	}

	// 蜜罐IP范围参数非空时执行有效性校验
	if cr.CanUseHoneyIPRange != "" {
		// 解析IP范围字符串为具体IP列表
		ipList, err1 := ip.ParseIPRange(cr.CanUseHoneyIPRange)
		if err1 != nil {
			response.FailWithMsg(err1.Error(), c)
			return
		}

		// 校验每个IP是否属于当前网络子网
		for _, s := range ipList {
			if !model.InSubnet(s) {
				response.FailWithMsg(fmt.Sprintf("%s不属于当前子网", s), c)
				return
			}
		}
	}

	// 更新网络信息到数据库
	err = global.DB.Model(&model).Updates(map[string]any{
		"title":                  cr.Title,
		"gateway":                cr.Gateway,
		"can_use_honey_ip_range": cr.CanUseHoneyIPRange,
	}).Error
	if err != nil {
		response.FailWithMsg("网络信息修改失败", c)
		return
	}

	response.OkWithMsg("网络信息修改成功", c)
}
