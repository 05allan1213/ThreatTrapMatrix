package honey_ip_api

// File: honey_server/api/honey_ip_api/remove.go
// Description: 诱捕IP删除API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/service/mq_service"
	"honey_server/internal/service/redis_service/net_lock"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// RemoveView 处理诱捕IP批量删除请求，包含节点状态校验与状态更新
func (HoneyIPApi) RemoveView(c *gin.Context) {
	log := middleware.GetLog(c)
	// 获取批量删除请求的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)

	log.WithFields(map[string]interface{}{
		"honey_ip_ids": cr.IdList,
	}).Info("honey IP deletion request received") // 收到诱捕IP批量删除请求

	// 查询待删除的诱捕IP列表，并预加载关联的节点信息
	var honeyIPList []models.HoneyIpModel
	if err := global.DB.Preload("NodeModel").Preload("NetModel").Find(&honeyIPList, "id in ?", cr.IdList).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"honey_ip_ids": cr.IdList,
			"error":        err,
		}).Error("failed to fetch honey IPs") // 获取诱捕IP列表失败
		response.FailWithMsg("查询诱捕IP失败", c)
		return
	}

	// 批量删除的子网数量
	var netMap = map[uint]any{}
	for _, model := range honeyIPList {
		netMap[model.NetID] = struct{}{}
	}
	if len(netMap) > 1 {
		response.FailWithMsg("批量删除仅支持单个子网", c)
		return
	}

	netModel := honeyIPList[0].NetModel

	// 检查是否存在指定的诱捕IP记录
	if len(honeyIPList) == 0 {
		log.WithFields(map[string]interface{}{
			"honey_ip_ids": cr.IdList,
		}).Warn("no honey IPs found") // 未找到诱捕IP
		response.FailWithMsg("未找到诱捕ip", c)
		return
	}

	// 检查所有诱捕IP是否属于同一节点
	nodeUID := honeyIPList[0].NodeModel.Uid
	for _, honeyIP := range honeyIPList {
		if honeyIP.NodeModel.Uid != nodeUID {
			log.WithFields(map[string]interface{}{
				"honey_ip_ids": cr.IdList,
				"mixed_nodes":  true,
			}).Warn("honey IPs belong to different nodes") // 诱捕IP属于不同节点
			response.FailWithMsg("批量删除的诱捕IP必须属于同一节点", c)
			return
		}
	}

	// 获取关联的节点信息（取第一个诱捕IP所属节点）
	nodeModel := honeyIPList[0].NodeModel

	// 校验节点是否处于运行状态
	if nodeModel.Status != 1 {
		log.WithFields(map[string]interface{}{
			"node_id": nodeModel.ID,
			"status":  nodeModel.Status,
		}).Warn("node is not running") // 节点未运行
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 校验节点是否在线（可通信）
	_, ok := grpc_service.GetNodeCommand(nodeModel.Uid)
	if !ok {
		log.WithFields(map[string]interface{}{
			"node_uid": nodeModel.Uid,
		}).Warn("node is offline") // 节点离线
		response.FailWithMsg("节点离线中", c)
		return
	}

	err := net_lock.Lock(netModel.ID)
	if err != nil {
		response.FailWithMsg("当前子网正在操作中", c)
		return
	}

	// 发送删除IP消息给节点
	req := mq_service.DeleteIPRequest{
		LogID: log.Data["logID"].(string),
		NetID: netModel.ID,
	}
	for _, model := range honeyIPList {
		isTan := model.NetModel.IP == model.IP
		req.IpList = append(req.IpList, mq_service.IpInfo{
			HoneyIPID: model.ID,
			IP:        model.IP,
			Network:   model.Network,
			IsTan:     isTan,
		})
	}
	log.WithFields(map[string]interface{}{
		"node_uid": nodeModel.Uid,
		"ip_count": len(req.IpList),
	}).Info("sending batch delete request to node") // 发送批量删除请求给节点

	mq_service.SendDeleteIPMsg(nodeModel.Uid, req)

	// 更新诱捕IP状态为删除中
	if err := global.DB.Model(&honeyIPList).Update("status", 4).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"node_uid":   nodeModel.Uid,
			"ip_count":   len(honeyIPList),
			"new_status": 4,
			"error":      err,
		}).Error("failed to update honey IP status") // 数据库更新诱捕IP状态失败
		response.FailWithMsg("更新诱捕IP状态失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"node_uid": nodeModel.Uid,
		"ip_count": len(honeyIPList),
	}).Info("batch deletion initiated successfully") // 批量删除成功启动

	mq_service.SendWsMsg(mq_service.WsMsgType{
		Type:   1,
		NetID:  netModel.ID,
		NodeID: netModel.NodeID,
	})

	// 返回删除处理中的响应
	response.OkWithMsg("批量删除中", c)
}
