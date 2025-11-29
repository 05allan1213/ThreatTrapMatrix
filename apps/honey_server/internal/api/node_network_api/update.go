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
	// 绑定并验证请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	var model models.NodeNetworkModel
	// 查询指定ID的网卡记录
	err := global.DB.Take(&model, cr.ID).Error
	if err != nil {
		response.FailWithMsg("节点网卡不存在", c)
		return
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

		// 验证网关属于当前网卡所在子网
		_, _net, _ := net.ParseCIDR(fmt.Sprintf("%s/%d", model.IP, model.Mask))
		if !_net.Contains(gateway) {
			response.FailWithMsg("网关ip不属于当前子网", c)
			return
		}
	}

	// 更新网卡网关配置
	err = global.DB.Model(&model).Update("gateway", cr.Gateway).Error
	if err != nil {
		response.FailWithMsg("节点网卡修改失败", c)
		return
	}

	response.OkWithMsg("节点网卡修改成功", c)
}
