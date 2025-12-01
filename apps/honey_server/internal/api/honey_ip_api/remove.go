package honey_ip_api

// File: honey_server/api/honey_ip_api/remove.go
// Description: 诱捕IP删除API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/service/mq_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// RemoveView 处理诱捕IP批量删除请求，包含节点状态校验与状态更新
func (HoneyIPApi) RemoveView(c *gin.Context) {
	// 获取批量删除请求的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)

	// 查询待删除的诱捕IP列表，并预加载关联的节点信息
	var honeyIPList []models.HoneyIpModel
	global.DB.Preload("NodeModel").Preload("NetModel").Find(&honeyIPList, "id in ?", cr.IdList)

	// 检查是否存在指定的诱捕IP记录
	if len(honeyIPList) == 0 {
		response.FailWithMsg("未找到诱捕ip", c)
		return
	}

	// 获取关联的节点信息（取第一个诱捕IP所属节点）
	nodeModel := honeyIPList[0].NodeModel

	// 校验节点是否处于运行状态
	if nodeModel.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 校验节点是否在线（可通信）
	_, ok := grpc_service.GetNodeCommand(nodeModel.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 发送删除IP消息给节点
	req := mq_service.DeleteIPRequest{
		LogID: "",
	}
	for _, model := range honeyIPList {
		var isTan bool
		if model.NetModel.IP == model.IP {
			isTan = true
		}
		req.IpList = append(req.IpList, mq_service.IpInfo{
			HoneyIPID: model.ID,
			IP:        model.IP,
			Network:   model.Network,
			IsTan:     isTan,
		})
	}
	mq_service.SendDeleteIPMsg(nodeModel.Uid, req)

	// 更新诱捕IP状态为删除中
	global.DB.Model(&honeyIPList).Update("status", 4)

	// 返回删除处理中的响应
	response.OkWithMsg("批量删除中", c)
}
