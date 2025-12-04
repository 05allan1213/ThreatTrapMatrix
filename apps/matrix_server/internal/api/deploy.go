package api

// File: matrix_server/api/deploy.go
// Description: 实现诱捕IP批量部署API接口

import (
	"fmt"
	"matrix_server/internal/global"
	"matrix_server/internal/middleware"
	"matrix_server/internal/models"
	"matrix_server/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// IpInfo 单个部署IP的信息结构体
type IpInfo struct {
	Ip             string `json:"ip" binding:"required,ip"` // 待部署的IP地址
	HostTemplateID uint   `json:"hostTemplateID"`           // 关联的主机模板ID
}

// DeployRequest 诱捕IP批量部署请求参数结构体
type DeployRequest struct {
	List  []IpInfo `json:"list" binding:"required,dive,required"` // 待部署IP列表
	NetID uint     `json:"netID" binding:"required"`              // 子网ID
}

// DeployResponse 诱捕IP批量部署响应结构体
type DeployResponse struct {
	DeployID string `json:"deployID"` // 部署id，用于标识本次部署任务
}

// DispatchDeployRequest 下发到节点的部署请求结构体
type DispatchDeployRequest struct {
	Ip       string     `json:"ip"`       // 待部署的IP地址
	PortList []PortInfo `json:"portList"` // 该IP对应的端口转发配置列表
}

// PortInfo 端口转发配置信息结构体
type PortInfo struct {
	Ip       string `json:"ip"`       // 源IP地址
	Port     int    `json:"port"`     // 源端口
	DestIp   string `json:"destIp"`   // 目标IP地址
	DestPort int    `json:"destPort"` // 目标端口
}

// DeployView 诱捕IP批量部署接口处理函数
func (Api) DeployView(c *gin.Context) {
	// 绑定并解析部署请求参数
	cr := middleware.GetBind[DeployRequest](c)
	// 校验待部署IP列表不能为空
	if len(cr.List) == 0 {
		response.FailWithMsg("至少需要选择一个ip进行部署", c)
		return
	}

	// 查询指定子网信息，并预加载关联的节点信息
	var model models.NetModel
	err := global.DB.Preload("NodeModel").Take(&model, cr.NetID).Error
	if err != nil {
		response.FailWithMsg("子网不存在", c)
		return
	}

	// 1. 校验节点是否在线
	node := model.NodeModel
	if node.Status != 1 {
		response.FailWithMsg("节点离线", c)
		return
	}

	// 提取所有待使用的主机模板ID，用于批量查询模板信息
	var hostTemplateIDList []uint
	for _, info := range cr.List {
		if info.HostTemplateID != 0 {
			hostTemplateIDList = append(hostTemplateIDList, info.HostTemplateID)
		}
	}

	// 查询主机模板列表，并构建模板ID到模板信息的映射
	var hostTemplateList []models.HostTemplateModel
	global.DB.Find(&hostTemplateList, "id in ?", hostTemplateIDList)
	var hostTemplateMap = map[uint]models.HostTemplateModel{}

	// 提取主机模板关联的服务ID，用于批量查询服务信息
	var serviceIDList []uint
	for _, templateModel := range hostTemplateList {
		hostTemplateMap[templateModel.ID] = templateModel
		for _, port := range templateModel.PortList {
			serviceIDList = append(serviceIDList, port.ServiceID)
		}
	}

	// 查询服务列表，并构建服务ID到服务信息的映射
	var serviceList []models.ServiceModel
	global.DB.Find(&serviceList, "id in ?", serviceIDList)
	var serviceMap = map[uint]models.ServiceModel{}
	for _, serviceModel := range serviceList {
		serviceMap[serviceModel.ID] = serviceModel
	}

	// 查询子网下已存在的资产IP，构建IP映射表用于合法性校验
	var hostMap = map[string]bool{}
	var assetsList []models.HostModel
	global.DB.Find(&assetsList, "net_id = ?", cr.NetID)
	for _, hostModel := range assetsList {
		hostMap[hostModel.IP] = true
	}

	// 查询子网下已存在的诱捕IP，构建IP映射表用于合法性校验
	var honeIpMap = map[string]bool{}
	var honeyIpList []models.HoneyIpModel
	global.DB.Find(&honeyIpList, "net_id = ?", cr.NetID)
	for _, honeyModel := range honeyIpList {
		honeIpMap[honeyModel.IP] = true
	}

	// 初始化下发部署的请求列表、待创建的诱捕IP列表、待创建的诱捕端口列表
	var list []DispatchDeployRequest
	var createHoneyIpList []models.HoneyIpModel
	var createHoneyPortList []models.HoneyPortModel

	// 2. 校验主机模板合法性，并逐行处理待部署IP信息
	for _, info := range cr.List {
		// 校验主机模板是否存在
		hostTemplateModel, ok := hostTemplateMap[info.HostTemplateID]
		if !ok {
			response.FailWithMsg(fmt.Sprintf("%d 主机模板不存在", info.HostTemplateID), c)
			return
		}

		// 3. 校验IP合法性：不能是已存在的资产IP
		if hostMap[info.Ip] {
			response.FailWithMsg(fmt.Sprintf("%s 是资产ip", info.Ip), c)
			return
		}
		// 校验IP合法性：不能是已存在的诱捕IP
		if honeIpMap[info.Ip] {
			response.FailWithMsg(fmt.Sprintf("%s 是诱捕ip", info.Ip), c)
			return
		}

		// 4. 待补充校验：IP不能是部署中/删除中的IP

		// 解析主机模板的端口列表，构建端口转发配置
		var portList []PortInfo
		for _, port := range hostTemplateModel.PortList {
			// 校验端口关联的服务是否存在
			service, ok1 := serviceMap[port.ServiceID]
			if !ok1 {
				response.FailWithMsg(
					fmt.Sprintf("主机模板%s %d 虚拟服务不存在",
						hostTemplateModel.Title, port.ServiceID), c)
				return
			}

			// 构建端口转发配置信息
			portList = append(portList, PortInfo{
				Ip:       info.Ip,
				Port:     port.Port,
				DestIp:   service.IP,
				DestPort: service.Port,
			})

			// 构建待入库的诱捕端口数据
			createHoneyPortList = append(createHoneyPortList, models.HoneyPortModel{
				NodeID:    model.NodeID,
				NetID:     cr.NetID,
				ServiceID: port.ServiceID,
				IP:        info.Ip,
				Port:      port.Port,
				DstIP:     service.IP,
				DstPort:   service.Port,
				Status:    1,
			})
		}

		// 构建下发部署的请求数据
		list = append(list, DispatchDeployRequest{
			Ip: info.Ip,
		})

		// 构建待入库的诱捕IP数据
		createHoneyIpList = append(createHoneyIpList, models.HoneyIpModel{
			NodeID: model.NodeID,
			NetID:  cr.NetID,
			IP:     info.Ip,
			Status: 1,
		})
	}

	// 5. 待补充逻辑：判断子网是否正在部署中
	// 待补充逻辑：分布式锁，锁住当前子网防止并发部署
	// 待补充逻辑：组装IP+端口转发数据

	// 批量创建诱捕IP数据，状态设为创建中
	err = global.DB.Create(&createHoneyIpList).Error
	if err != nil {
		response.FailWithMsg("批量部署失败", c)
		return
	}
	logrus.Infof("批量部署%d诱捕ip", len(createHoneyIpList))

	// 批量创建诱捕端口转发数据（如有）
	if len(createHoneyPortList) > 0 {
		err = global.DB.Create(&createHoneyPortList).Error
		if err != nil {
			response.FailWithMsg("批量部署失败", c)
			return
		}
		logrus.Infof("批量部署%d诱捕转发", len(createHoneyPortList))
	}

	// 待补充逻辑：组装部署数据，下发到MQ（一次批量部署仅下发一条消息）
	// 待优化点：若待部署IP数量过多，需拆分消息分批下发

	// 构建响应数据并返回成功结果
	data := DeployResponse{}
	response.OkWithData(data, c)
	return
}
