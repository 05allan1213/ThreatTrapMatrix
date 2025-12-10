package api

// File: matrix_server/api/deploy.go
// Description: 实现诱捕IP批量部署API接口

import (
	"errors"
	"fmt"
	"matrix_server/internal/global"
	"matrix_server/internal/middleware"
	"matrix_server/internal/models"
	"matrix_server/internal/service/mq_service"
	"matrix_server/internal/service/redis_service/net_lock"
	"matrix_server/internal/service/redis_service/net_progress"
	"matrix_server/internal/utils"
	"matrix_server/internal/utils/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// IpInfo 批量部署请求中的单个IP信息结构体
type IpInfo struct {
	Ip             string `json:"ip" binding:"required,ip"` // 待部署的IP地址
	HostTemplateID *uint  `json:"hostTemplateID"`           // 关联的主机模板ID
}

// DeployRequest 批量部署接口的请求参数结构体
type DeployRequest struct {
	List  []IpInfo `json:"list" binding:"required,dive,required"` // 待部署IP列表
	NetID uint     `json:"netID" binding:"required"`              // 子网ID
}

// DeployView 批量IP部署接口处理函数
func (Api) DeployView(c *gin.Context) {
	// 绑定并解析前端提交的批量部署请求参数
	cr := middleware.GetBind[DeployRequest](c)
	// 获取请求关联的日志实例（含traceID）
	log := middleware.GetLog(c)

	// 记录部署请求接收日志，包含子网ID和IP数量
	log.WithFields(map[string]interface{}{
		"net_id":   cr.NetID,
		"ip_count": len(cr.List),
	}).Info("batch deployment request received")

	// 校验待部署IP列表非空
	if len(cr.List) == 0 {
		log.Warn("no IPs selected for deployment")
		response.FailWithMsg("需要选择一个ip进行部署", c)
		return
	}

	// 查询子网信息并预加载关联的节点信息，校验子网是否存在
	var model models.NetModel
	if err := global.DB.Preload("NodeModel").Take(&model, cr.NetID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("subnet not found")
		response.FailWithMsg("子网不存在", c)
		return
	}

	// 检查节点在线状态（状态1为在线）
	node := model.NodeModel
	if node.Status != 1 {
		log.WithFields(map[string]interface{}{
			"node_id":  node.ID,
			"node_uid": node.Uid,
			"status":   node.Status,
		}).Warn("node is offline")
		response.FailWithMsg("节点离线", c)
		return
	}

	// 收集待部署IP关联的唯一主机模板ID（去重）
	var hostTemplateIDList []uint
	for _, info := range cr.List {
		if info.HostTemplateID != nil {
			// 过滤空模板ID，且仅添加未存在的模板ID
			if (*info.HostTemplateID) != 0 && !utils.InList(hostTemplateIDList, *info.HostTemplateID) {
				hostTemplateIDList = append(hostTemplateIDList, *info.HostTemplateID)
			}
		}
	}

	// 加载主机模板及关联的端口列表（仅当有模板ID时）
	var hostTemplateList []models.HostTemplateModel
	if len(hostTemplateIDList) > 0 {
		if err := global.DB.Find(&hostTemplateList, "id in ?", hostTemplateIDList).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"template_ids": hostTemplateIDList,
				"error":        err,
			}).Error("failed to load host templates")
			response.FailWithMsg("加载主机模板失败", c)
			return
		}
	}

	// 构建主机模板ID到模板实例的映射，便于快速查询；同时收集模板关联的服务ID
	hostTemplateMap := make(map[uint]models.HostTemplateModel)
	var serviceIDList []uint
	for _, templateModel := range hostTemplateList {
		hostTemplateMap[templateModel.ID] = templateModel
		for _, port := range templateModel.PortList {
			// 收集唯一的服务ID，避免重复加载
			if !utils.InList(serviceIDList, port.ServiceID) {
				serviceIDList = append(serviceIDList, port.ServiceID)
			}
		}
	}

	// 加载模板关联的虚拟服务信息（仅当有服务ID时）
	var serviceList []models.ServiceModel
	if len(serviceIDList) > 0 {
		if err := global.DB.Find(&serviceList, "id in ?", serviceIDList).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"service_ids": serviceIDList,
				"error":       err,
			}).Error("failed to load services")
			response.FailWithMsg("加载服务信息失败", c)
			return
		}
	}

	// 构建服务ID到服务实例的映射，便于快速查询
	serviceMap := make(map[uint]models.ServiceModel)
	for _, serviceModel := range serviceList {
		serviceMap[serviceModel.ID] = serviceModel
	}

	// 加载子网下已存在的主机资产，用于IP冲突校验（避免部署资产IP）
	var assetsList []models.HostModel
	if err := global.DB.Find(&assetsList, "net_id = ?", cr.NetID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Error("failed to load existing hosts")
		response.FailWithMsg("查询资产信息失败", c)
		return
	}

	// 构建资产IP映射，快速判断IP是否为已存在的资产
	hostMap := make(map[string]bool)
	for _, hostModel := range assetsList {
		hostMap[hostModel.IP] = true
	}

	// 加载子网下已存在的诱捕IP，用于IP冲突校验（避免重复部署）
	var honeyIpList []models.HoneyIpModel
	if err := global.DB.Find(&honeyIpList, "net_id = ?", cr.NetID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Error("failed to load existing honey IPs")
		response.FailWithMsg("查询诱捕IP信息失败", c)
		return
	}

	// 构建诱捕IP到实例的映射，快速判断IP是否为已存在的诱捕IP及状态
	honeIpMap := make(map[string]models.HoneyIpModel)
	for _, honeyModel := range honeyIpList {
		honeIpMap[honeyModel.IP] = honeyModel
	}

	// 获取日志追踪ID，用于关联部署全流程日志
	logID := log.Data["logID"].(string)
	// 构建MQ批量部署消息结构体
	var batchDeployData = mq_service.BatchDeployRequest{
		NetID:   cr.NetID,
		LogID:   logID,
		Network: model.Network,
		TanIp:   model.IP,
	}

	// 待创建的诱捕IP记录列表
	var createHoneyIpList []models.HoneyIpModel
	// 待创建的诱捕端口记录列表
	var createHoneyPortList []models.HoneyPortModel

	// 遍历待部署IP列表，完成IP冲突校验并构建部署数据
	for _, info := range cr.List {
		// 校验IP是否为已存在的资产IP
		if hostMap[info.Ip] {
			log.WithFields(map[string]interface{}{
				"ip": info.Ip,
			}).Warn("IP conflict with existing host")
			response.FailWithMsg(fmt.Sprintf("%s 是资产ip", info.Ip), c)
			return
		}

		// 校验IP是否为已存在的诱捕IP，并判断状态
		if honeyIpModel, exists := honeIpMap[info.Ip]; exists {
			// 状态1：部署中，禁止重复部署
			if honeyIpModel.Status == 1 {
				log.WithFields(map[string]interface{}{
					"ip":     info.Ip,
					"status": honeyIpModel.Status,
				}).Warn("IP is already deploying")
				response.FailWithMsg(fmt.Sprintf("%s 正在部署中", info.Ip), c)
				return
			}
			// 状态2：已部署完成，禁止重复部署
			if honeyIpModel.Status == 2 {
				log.WithFields(map[string]interface{}{
					"ip":     info.Ip,
					"status": honeyIpModel.Status,
				}).Warn("IP is already a honey IP")
				response.FailWithMsg(fmt.Sprintf("%s 是诱捕ip", info.Ip), c)
				return
			}
		}

		// 构建端口转发信息（仅当关联主机模板时）
		var portList []mq_service.PortInfo
		if info.HostTemplateID != nil {
			// 校验主机模板是否存在
			hostTemplateModel, ok := hostTemplateMap[*info.HostTemplateID]
			if !ok {
				log.WithFields(map[string]interface{}{
					"template_id": info.HostTemplateID,
					"ip":          info.Ip,
				}).Warn("host template not found")
				response.FailWithMsg(fmt.Sprintf("%d 主机模板不存在", info.HostTemplateID), c)
				return
			}

			// 遍历模板关联的端口，构建端口转发信息及诱捕端口记录
			for _, port := range hostTemplateModel.PortList {
				// 校验模板关联的服务是否存在
				service, ok1 := serviceMap[port.ServiceID]
				if !ok1 {
					log.WithFields(map[string]interface{}{
						"template_id": hostTemplateModel.ID,
						"service_id":  port.ServiceID,
					}).Warn("service not found for template port")
					response.FailWithMsg(
						fmt.Sprintf("主机模板%s %d 虚拟服务不存在",
							hostTemplateModel.Title, port.ServiceID), c)
					return
				}

				// 构建MQ端口转发信息
				portInfo := mq_service.PortInfo{
					IP:       info.Ip,
					Port:     port.Port,
					DestIP:   service.IP,
					DestPort: service.Port,
				}
				portList = append(portList, portInfo)

				// 构建待创建的诱捕端口记录
				createHoneyPortList = append(createHoneyPortList, models.HoneyPortModel{
					NodeID:    model.NodeID,
					NetID:     cr.NetID,
					ServiceID: port.ServiceID,
					IP:        info.Ip,
					Port:      port.Port,
					DstIP:     service.IP,
					DstPort:   service.Port,
					Status:    1, // 状态1：部署中
				})
			}
		}

		// 将当前IP的部署信息添加到批量部署数据中
		batchDeployData.IPList = append(batchDeployData.IPList, mq_service.DeployIp{
			Ip:       info.Ip,
			Mask:     model.Mask,
			PortList: portList,
		})

		// 构建待创建的诱捕IP记录
		createHoneyIpList = append(createHoneyIpList, models.HoneyIpModel{
			NodeID:         model.NodeID,
			NetID:          cr.NetID,
			IP:             info.Ip,
			HostTemplateID: info.HostTemplateID,
			Status:         1, // 状态1：部署中
		})
	}

	// 获取子网分布式锁，防止同子网并发部署
	if err := net_lock.Lock(cr.NetID); err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("failed to acquire network lock")
		response.FailWithMsg("当前子网正在部署中", c)
		return
	}

	// 执行数据库事务：创建诱捕IP/端口记录、设置部署进度、下发MQ部署消息
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 批量创建诱捕IP记录
		if err := tx.Create(&createHoneyIpList).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"error": err,
			}).Error("failed to create honey IPs")
			return errors.New("批量部署失败")
		}
		log.WithFields(map[string]interface{}{
			"created_ips": len(createHoneyIpList),
		}).Info("honey IPs created")

		// 批量创建诱捕端口记录（仅当有端口记录时）
		if len(createHoneyPortList) > 0 {
			// 构建诱捕IP到ID的映射，关联端口记录的HoneyIpID
			honeyIpToIDMap := make(map[string]uint)
			for _, ipModel := range createHoneyIpList {
				honeyIpToIDMap[ipModel.IP] = ipModel.ID
			}

			var createPortList []models.HoneyPortModel
			for _, portModel := range createHoneyPortList {
				portModel.HoneyIpID = honeyIpToIDMap[portModel.IP]
				createPortList = append(createPortList, portModel)
			}

			if err := tx.Create(&createPortList).Error; err != nil {
				log.WithFields(map[string]interface{}{
					"error": err,
				}).Error("failed to create honey ports")
				return errors.New("批量部署失败")
			}
			log.WithFields(map[string]interface{}{
				"created_ports": len(createPortList),
			}).Info("honey ports created")
		}

		// 设置子网部署进度（类型1：部署，总数量为待部署IP数）
		if err := net_progress.Set(cr.NetID, net_progress.NetDeployInfo{
			Type:     1,
			AllCount: int64(len(cr.List)),
		}); err != nil {
			log.WithFields(map[string]interface{}{
				"net_id": cr.NetID,
				"error":  err,
			}).Error("failed to set deployment progress")
			return errors.New("设置操作进度失败")
		}

		// 下发批量部署MQ消息到节点
		if err := mq_service.SendBatchDeployMsg(node.Uid, batchDeployData); err != nil {
			log.WithFields(map[string]interface{}{
				"node_uid": node.Uid,
				"error":    err,
			}).Error("failed to send deployment message")
			return errors.New("部署消息下发失败")
		}

		return nil
	})

	// 处理事务执行结果
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("deployment transaction failed")
		net_lock.UnLock(cr.NetID) // 事务失败释放分布式锁
		response.FailWithError(err, c)
		return
	}

	// 记录部署启动成功日志
	log.WithFields(map[string]interface{}{
		"net_id":       cr.NetID,
		"deployed_ips": len(createHoneyIpList),
	}).Info("batch deployment initiated successfully")

	// 推送WebSocket部署通知（类型1：部署）
	mq_service.SendWsMsg(mq_service.WsMsgType{
		Type:   1,
		NetID:  cr.NetID,
		NodeID: node.ID,
	})

	// 返回部署启动成功响应
	response.OkWithMsg("批量部署成功，正在部署中", c)
}
