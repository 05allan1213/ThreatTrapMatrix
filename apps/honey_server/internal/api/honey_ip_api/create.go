package honey_ip_api

// File: honey_server/api/honey_ip_api/create.go
// Description: 诱捕IP创建API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/service/mq_service"
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
	// 获取并绑定创建请求参数
	cr := middleware.GetBind[CreateRequest](c)

	// 校验网络是否存在，并预加载关联的节点信息
	var netModel models.NetModel
	err := global.DB.Preload("NodeModel").Take(&netModel, cr.NetID).Error
	if err != nil {
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 校验IP是否属于当前网络的可用IP范围
	ipRange, err := netModel.IpRange()
	if !utils.InList(ipRange, cr.IP) {
		response.FailWithMsg("当前ip不存在可部署ip列表里面", c)
		return
	}

	// 校验IP是否已被真实主机占用
	var hostModel models.HostModel
	err = global.DB.Take(&hostModel, "net_id = ? and ip = ?", cr.NetID, cr.IP).Error
	if err == nil {
		response.FailWithMsg("当前ip是主机ip", c)
		return
	}

	// 校验IP是否已被部署为诱捕IP
	var honeyIPModel models.HoneyIpModel
	err = global.DB.Take(&honeyIPModel, "net_id = ? and ip = ?", cr.NetID, cr.IP).Error
	if err == nil {
		response.FailWithMsg("当前ip已使用", c)
		return
	}

	// 校验节点是否处于运行状态
	if netModel.NodeModel.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 校验节点是否在线（可通信）
	_, ok := grpc_service.GetNodeCommand(netModel.NodeModel.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 构建诱捕IP模型并写入数据库
	var model = models.HoneyIpModel{
		NodeID: netModel.NodeID, // 所属节点ID
		NetID:  netModel.ID,     // 所属网络ID
		IP:     cr.IP,           // 诱捕IP地址
		Status: 1,               // 状态：启用
	}
	err = global.DB.Create(&model).Error
	if err != nil {
		response.FailWithMsg("创建诱捕ip失败", c)
		return
	}

	// 判断当前IP是否是探针IP
	var isTan bool
	if netModel.IP == model.IP {
		isTan = true
	}

	//  发送创建IP消息给节点
	mq_service.SendCreateIPMsg(netModel.NodeModel.Uid, mq_service.CreateIPRequest{
		HoneyIPID: model.ID,
		IP:        model.IP,
		Mask:      netModel.Mask,
		Network:   netModel.Network,
		IsTan:     isTan,
	})

	// 返回创建成功的诱捕IP ID
	response.OkWithData(model.ID, c)
}
