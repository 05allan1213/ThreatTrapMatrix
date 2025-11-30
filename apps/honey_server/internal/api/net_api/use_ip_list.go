package net_api

// File: honey_server/api/net_api/use_ip_list.go
// Description: 网络可用IP列表查询API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/ip"
	"honey_server/internal/utils/response"
	"net"

	"github.com/gin-gonic/gin"
)

// NetUseIPListResponse 网络IP使用情况响应结构体
type NetUseIPListResponse struct {
	Total              int      `json:"total"`              // 子网总IP数量
	Used               int      `json:"used"`               // 已使用IP数量
	UseIPList          []string `json:"useIPList"`          // 可用IP列表
	CanUseHoneyIPRange string   `json:"canUseHoneyIPRange"` // 子网内可分配的诱捕IP范围
}

// NetUseIPListView 查询网络可用IP列表与使用状态
func (NetApi) NetUseIPListView(c *gin.Context) {
	// 获取请求绑定的网络ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	var model models.NetModel
	// 查询网络基础信息
	err := global.DB.Take(&model, cr.Id).Error
	if err != nil {
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 若未计算过可用IP范围，则自动计算
	if model.CanUseHoneyIPRange == "" {
		// 解析子网CIDR格式
		_, ipNet, err := net.ParseCIDR(model.Subnet())
		if err != nil {
			response.FailWithMsg("无效的子网格式", c)
			return
		}

		// 计算可用IP范围（排除网络地址和广播地址）
		startIP := ip.IncrementIP(ipNet.IP)            // 子网起始可用IP
		endIP := ip.DecrementIP(ip.BroadcastIP(ipNet)) // 子网结束可用IP
		model.CanUseHoneyIPRange = ip.FormatIPRange(startIP, endIP)
	}

	// 解析可分配IP范围为具体IP列表
	ipList, err := ip.ParseIPRange(model.CanUseHoneyIPRange)
	if err != nil {
		response.FailWithMsg("解析IP范围失败", c)
		return
	}

	// 查询已占用的IP列表（主机IP与诱捕IP）
	var filterIPList1, filterIPList2 []string
	global.DB.Model(models.HostModel{}).Where("net_id = ?", cr.Id).Select("ip").Scan(&filterIPList1)
	global.DB.Model(models.HoneyIpModel{}).Where("net_id = ?", cr.Id).Select("ip").Scan(&filterIPList2)

	// 合并已使用IP，去重存储
	usedIPs := make(map[string]struct{})
	for _, ip := range filterIPList1 {
		usedIPs[ip] = struct{}{}
	}
	for _, ip := range filterIPList2 {
		usedIPs[ip] = struct{}{}
	}

	// 筛选出未被使用的可用IP
	var availableIPs []string
	for _, ip := range ipList {
		if _, exists := usedIPs[ip]; !exists {
			availableIPs = append(availableIPs, ip)
		}
	}

	// 返回IP使用统计与可用列表
	response.OkWithData(NetUseIPListResponse{
		Total:              len(ipList),
		Used:               len(ipList) - len(availableIPs),
		UseIPList:          availableIPs,
		CanUseHoneyIPRange: model.CanUseHoneyIPRange,
	}, c)
}
