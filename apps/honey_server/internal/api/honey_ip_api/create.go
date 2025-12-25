package honey_ip_api

// File: honey_server/api/honey_ip_api/create.go
// Description: 诱捕IP创建API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/service/mq_service"
	"honey_server/internal/service/redis_service/net_lock"
	"honey_server/internal/utils"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// CreateRequest 诱捕IP创建请求参数结构体
type CreateRequest struct {
	NetID uint   `json:"netID" binding:"required"` // 所属网络ID（必填）
	IP    string `json:"ip" binding:"required"`    // 诱捕IP地址（必填）
}

// CreateView 处理诱捕IP创建请求，包含多重前置校验逻辑
func (HoneyIPApi) CreateView(c *gin.Context) {
	log := middleware.GetLog(c)

	// 获取并绑定创建请求参数
	cr := middleware.GetBind[CreateRequest](c)

	log.WithFields(map[string]interface{}{
		"net_id": cr.NetID,
		"ip":     cr.IP,
	}).Info("honey IP creation request received") // 收到诱捕IP创建请求

	// 校验网络是否存在，并预加载关联的节点信息
	var netModel models.NetModel
	if err := global.DB.Preload("NodeModel").Take(&netModel, cr.NetID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("network not found") // 未找到网络
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 校验IP是否属于当前网络的可用IP范围
	ipRange, err := netModel.IpRange()
	if err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": netModel.ID,
			"error":  err,
		}).Error("failed to get network IP range") // 获取网络IP范围失败
		response.FailWithMsg("获取网络IP范围失败", c)
		return
	}

	// 校验IP是否在可部署IP列表中
	if !utils.InList(ipRange, cr.IP) {
		log.WithFields(map[string]interface{}{
			"net_id":   netModel.ID,
			"ip":       cr.IP,
			"ip_range": ipRange,
		}).Warn("IP not in deployable range") // ip不在可部署ip列表里面
		response.FailWithMsg("当前ip不存在可部署ip列表里面", c)
		return
	}

	// 校验IP是否已被真实主机占用
	var hostModel models.HostModel
	if err := global.DB.Take(&hostModel, "net_id = ? and ip = ?", cr.NetID, cr.IP).Error; err == nil {
		log.WithFields(map[string]interface{}{
			"net_id":  cr.NetID,
			"ip":      cr.IP,
			"host_id": hostModel.ID,
		}).Warn("IP is a host IP") // ip是主机ip
		response.FailWithMsg("当前ip是主机ip", c)
		return
	}

	// 校验IP是否已被部署为诱捕IP
	var honeyIPModel models.HoneyIpModel
	if err := global.DB.Take(&honeyIPModel, "net_id = ? and ip = ?", cr.NetID, cr.IP).Error; err == nil {
		log.WithFields(map[string]interface{}{
			"net_id":      cr.NetID,
			"ip":          cr.IP,
			"honey_ip_id": honeyIPModel.ID,
		}).Warn("IP is already used as honey IP") // 当前ip已被用作诱捕IP
		response.FailWithMsg("当前ip已使用", c)
		return
	}

	// 校验节点是否处于运行状态
	if netModel.NodeModel.Status != 1 {
		log.WithFields(map[string]interface{}{
			"net_id":  netModel.ID,
			"node_id": netModel.NodeModel.ID,
			"status":  netModel.NodeModel.Status,
		}).Warn("node is not running") // 节点未运行
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 校验节点是否在线（可通信）
	_, ok := grpc_service.GetNodeCommand(netModel.NodeModel.Uid)
	if !ok {
		log.WithFields(map[string]interface{}{
			"node_uid": netModel.NodeModel.Uid,
			"node_id":  netModel.NodeModel.ID,
		}).Warn("node is offline") // 节点离线
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 锁定网络
	err = net_lock.Lock(cr.NetID)
	if err != nil {
		response.FailWithMsg("当前子网正在操作中", c)
		return
	}

	// 构建诱捕IP模型并写入数据库
	model := models.HoneyIpModel{
		NodeID: netModel.NodeID, // 所属节点ID
		NetID:  netModel.ID,     // 所属网络ID
		IP:     cr.IP,           // 诱捕IP地址
		Status: 1,               // 状态：启用
	}
	if err := global.DB.Create(&model).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": netModel.ID,
			"ip":     cr.IP,
			"error":  err,
		}).Error("failed to create honey IP record") // 创建诱捕ip记录失败
		response.FailWithMsg("创建诱捕ip失败", c)
		return
	}

	// 判断当前IP是否是探针IP
	isTan := netModel.IP == model.IP

	//  发送创建IP消息到队列
	mq_service.SendCreateIPMsg(netModel.NodeModel.Uid, mq_service.CreateIPRequest{
		HoneyIPID: model.ID,
		IP:        model.IP,
		Mask:      netModel.Mask,
		Network:   netModel.Network,
		IsTan:     isTan,
		LogID:     log.Data["logID"].(string),
	})

	log.WithFields(map[string]interface{}{
		"honey_ip_id": model.ID,
		"ip":          model.IP,
		"net_id":      netModel.ID,
		"node_uid":    netModel.NodeModel.Uid,
	}).Info("honey IP created successfully") // 诱捕IP创建成功

	mq_service.SendWsMsg(mq_service.WsMsgType{
		Type:   1,
		NetID:  cr.NetID,
		NodeID: model.NodeID,
	})

	// 返回创建成功的诱捕IP ID
	response.OkWithData(model.ID, c)
}
