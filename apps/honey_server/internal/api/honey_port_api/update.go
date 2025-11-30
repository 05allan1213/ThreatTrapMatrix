package honey_port_api

// File: honey_server/api/honey_port_api/update.go
// Description: 诱捕转发更新API接口

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/service/mq_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// UpdateRequest 诱捕转发更新请求参数结构体
type UpdateRequest struct {
	HoneyIPID uint       `json:"honeyIpID" binding:"required"`     // 关联的诱捕ipID（必填）
	PortList  []PortType `json:"portList" binding:"dive,required"` // 端口配置列表（必填，需逐个验证）
}

// PortType 端口配置项结构体
type PortType struct {
	Port      int  `json:"port" binding:"required,min=1,max=65535"` // 端口号（必填，范围1-65535）
	ServiceID uint `json:"serviceID" binding:"required"`            // 关联的服务ID（必填）
}

// UpdateView 处理诱捕转发更新请求，实现端口配置的增量更新
func (HoneyPortApi) UpdateView(c *gin.Context) {
	// 获取并绑定端口更新请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	// 校验关联的诱捕IP是否存在
	var honeyIPModel models.HoneyIpModel
	err := global.DB.Preload("NodeModel").Take(&honeyIPModel, cr.HoneyIPID).Error
	if err != nil {
		response.FailWithMsg("不存在的诱捕ip", c)
		return
	}

	nodeModel := honeyIPModel.NodeModel
	// 判断节点是否在线
	if nodeModel.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 使用封装的获取节点函数
	_, ok := grpc_service.GetNodeCommand(nodeModel.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 查询当前诱捕IP已配置的端口列表
	var honeyPortList []models.HoneyPortModel
	global.DB.Find(&honeyPortList, "honey_ip_id = ?", cr.HoneyIPID)

	// 校验端口配置有效性：端口不重复 + 服务ID存在性
	portMap := map[int]struct{}{} // 用于检测端口重复
	serviceIDList := []uint{}     // 收集所有关联的服务ID
	for _, portType := range cr.PortList {
		serviceIDList = append(serviceIDList, portType.ServiceID)
		portMap[portType.Port] = struct{}{}
	}

	// 检查是否存在重复端口
	if len(portMap) != len(cr.PortList) {
		response.FailWithMsg("端口重复", c)
		return
	}

	// 查询所有关联的服务信息，验证服务ID有效性
	var serviceList []models.ServiceModel
	global.DB.Find(&serviceList, "id in ?", serviceIDList)
	serviceMap := map[uint]models.ServiceModel{}
	for _, model := range serviceList {
		serviceMap[model.ID] = model
	}

	// 对比现有端口与请求端口，计算新增/删除的端口配置
	// 1. 构建现有端口的映射表（端口号->端口模型）
	existingPorts := make(map[int]models.HoneyPortModel)
	for _, port := range honeyPortList {
		existingPorts[port.Port] = port
	}

	// 2. 筛选需要新增的端口(端口不存在)
	var newPorts []models.HoneyPortModel
	for _, reqPort := range cr.PortList {
		// 验证服务ID是否存在
		service, ok := serviceMap[reqPort.ServiceID]
		if !ok {
			response.FailWithMsg(fmt.Sprintf("服务%d不存在", reqPort.ServiceID), c)
			return
		}

		// 端口不存在则加入新增列表
		if _, exists := existingPorts[reqPort.Port]; !exists {
			newPorts = append(newPorts, models.HoneyPortModel{
				HoneyIpID: cr.HoneyIPID,
				Port:      reqPort.Port,
				ServiceID: reqPort.ServiceID,
				DstIP:     service.IP,   // 从服务配置获取目标IP
				DstPort:   service.Port, // 从服务配置获取目标端口
				Status:    1,            // 启用状态
			})
		}
	}

	// 3. 筛选需要删除的端口(端口存在、请求中无)
	var portsToDelete []models.HoneyPortModel
	for port, model := range existingPorts {
		found := false
		for _, reqPort := range cr.PortList {
			if reqPort.Port == port {
				found = true
				break
			}
		}
		if !found {
			portsToDelete = append(portsToDelete, model)
		}
	}

	// 使用事务执行端口配置的增删操作，保证数据一致性
	tx := global.DB.Begin()
	if tx.Error != nil {
		response.FailWithMsg("更新端口信息失败", c)
		return
	}

	// 删除需要移除的端口
	for _, port := range portsToDelete {
		if err := tx.Delete(&port).Error; err != nil {
			tx.Rollback()
			response.FailWithMsg("更新端口信息失败", c)
			return
		}
	}

	// 添加新增的端口配置
	for _, port := range newPorts {
		if err := tx.Create(&port).Error; err != nil {
			tx.Rollback()
			response.FailWithMsg("更新端口信息失败", c)
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		response.FailWithMsg("更新端口信息失败", c)
		return
	}

	// 返回更新结果
	msg := fmt.Sprintf("新增端口%d个，删除端口%d个", len(newPorts), len(portsToDelete))

	// 发送端口绑定消息
	var portList []models.HoneyPortModel
	global.DB.Find(&portList, "honey_ip_id = ?", cr.HoneyIPID)
	req := mq_service.BindPortRequest{
		IP:    honeyIPModel.IP,
		LogID: "",
	}
	for _, model := range portList {
		req.PortList = append(req.PortList, mq_service.PortInfo{
			IP:       honeyIPModel.IP,
			Port:     model.Port,
			DestIP:   model.DstIP,
			DestPort: model.DstPort,
		})
	}
	mq_service.SendBindPortMsg(nodeModel.Uid, req)

	response.OkWithMsg(msg, c)
}
