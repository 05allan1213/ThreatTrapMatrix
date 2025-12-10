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
	HostTemplateID uint   `json:"hostTemplateID"`           // 关联的主机模板ID
}

// DeployRequest 批量部署接口的请求参数结构体
type DeployRequest struct {
	List  []IpInfo `json:"list" binding:"required,dive,required"` // 待部署IP列表
	NetID uint     `json:"netID" binding:"required"`              // 子网ID
}

// DeployView 批量部署接口处理函数
func (Api) DeployView(c *gin.Context) {
	// 绑定并解析请求参数到DeployRequest结构体
	cr := middleware.GetBind[DeployRequest](c)
	log := middleware.GetLog(c)

	log.WithFields(map[string]interface{}{
		"net_id":   cr.NetID,
		"ip_count": len(cr.List),
	}).Info("batch deployment request received") // 收到批量部署请求
	// 校验待部署IP列表是否为空
	if len(cr.List) == 0 {
		log.Warn("no IPs selected for deployment") // 没有选择IP进行部署
		response.FailWithMsg("需要选择一个ip进行部署", c)
		return
	}

	// 查询子网信息并预加载关联的节点信息
	var model models.NetModel
	if err := global.DB.Preload("NodeModel").Take(&model, cr.NetID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("subnet not found") // 子网不存在
		response.FailWithMsg("子网不存在", c)
		return
	}
	// 1. 校验节点在线状态（状态1表示在线）
	node := model.NodeModel
	if node.Status != 1 {
		log.WithFields(map[string]interface{}{
			"node_id":  node.ID,
			"node_uid": node.Uid,
			"status":   node.Status,
		}).Warn("node is offline") // 节点未运行
		response.FailWithMsg("节点离线", c)
		return
	}

	// 提取请求中所有非空的主机模板ID
	var hostTemplateIDList []uint
	for _, info := range cr.List {
		if info.HostTemplateID != 0 && !utils.InList(hostTemplateIDList, info.HostTemplateID) {
			hostTemplateIDList = append(hostTemplateIDList, info.HostTemplateID)
		}
	}
	// 查询主机模板列表并构建模板ID到模板的映射
	var hostTemplateList []models.HostTemplateModel
	if len(hostTemplateIDList) > 0 {
		if err := global.DB.Find(&hostTemplateList, "id in ?", hostTemplateIDList).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"template_ids": hostTemplateIDList,
				"error":        err,
			}).Error("failed to load host templates") // 加载主机模板失败
			response.FailWithMsg("加载主机模板失败", c)
			return
		}
	}

	hostTemplateMap := make(map[uint]models.HostTemplateModel)

	// 提取主机模板关联的服务ID列表
	var serviceIDList []uint
	for _, templateModel := range hostTemplateList {
		hostTemplateMap[templateModel.ID] = templateModel
		for _, port := range templateModel.PortList {
			if !utils.InList(serviceIDList, port.ServiceID) {
				serviceIDList = append(serviceIDList, port.ServiceID)
			}
		}
	}
	// 查询服务列表并构建服务ID到服务的映射
	var serviceList []models.ServiceModel
	if len(serviceIDList) > 0 {
		if err := global.DB.Find(&serviceList, "id in ?", serviceIDList).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"service_ids": serviceIDList,
				"error":       err,
			}).Error("failed to load services") // 加载服务信息失败
			response.FailWithMsg("加载服务信息失败", c)
			return
		}
	}

	serviceMap := make(map[uint]models.ServiceModel)
	for _, serviceModel := range serviceList {
		serviceMap[serviceModel.ID] = serviceModel
	}

	// 构建子网下资产IP的映射（用于校验IP唯一性）
	var assetsList []models.HostModel
	if err := global.DB.Find(&assetsList, "net_id = ?", cr.NetID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Error("failed to load existing hosts") // 获取子网下资产IP列表失败
		response.FailWithMsg("查询资产信息失败", c)
		return
	}

	hostMap := make(map[string]bool)
	for _, hostModel := range assetsList {
		hostMap[hostModel.IP] = true
	}

	// 构建子网下诱捕IP的映射（用于校验IP部署状态）
	var honeyIpList []models.HoneyIpModel
	if err := global.DB.Find(&honeyIpList, "net_id = ?", cr.NetID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Error("failed to load existing honey IPs") // 加载现有诱捕IP信息失败
		response.FailWithMsg("查询诱捕IP信息失败", c)
		return
	}

	honeIpMap := make(map[string]models.HoneyIpModel)
	for _, honeyModel := range honeyIpList {
		honeIpMap[honeyModel.IP] = honeyModel
	}

	// 获取上下文日志实例及日志ID
	logID := log.Data["logID"].(string)
	// 组装MQ批量部署请求数据
	var batchDeployData = mq_service.BatchDeployRequest{
		NetID:   cr.NetID,
		LogID:   logID,
		Network: model.Network,
		TanIp:   model.IP,
	}
	// 待入库的诱捕IP列表
	var createHoneyIpList []models.HoneyIpModel
	// 待入库的诱捕端口列表
	var createHoneyPortList []models.HoneyPortModel

	// 2. 遍历IP列表，校验主机模板合法性及IP唯一性
	for _, info := range cr.List {
		// 校验主机模板是否存在
		hostTemplateModel, ok := hostTemplateMap[info.HostTemplateID]
		if !ok {
			log.WithFields(map[string]interface{}{
				"template_id": info.HostTemplateID,
				"ip":          info.Ip,
			}).Warn("host template not found") // 主机模板不存在
			response.FailWithMsg(fmt.Sprintf("%d 主机模板不存在", info.HostTemplateID), c)
			return
		}
		// 3. 校验IP是否为资产IP/已部署诱捕IP
		if hostMap[info.Ip] {
			log.WithFields(map[string]interface{}{
				"ip": info.Ip,
			}).Warn("IP conflict with existing host") // IP与现有主机冲突
			response.FailWithMsg(fmt.Sprintf("%s 是资产ip", info.Ip), c)
			return
		}
		// 4. 校验IP是否为已部署诱捕IP
		if honeyIpModel, exists := honeIpMap[info.Ip]; exists {
			// 状态1：IP正在部署中
			if honeyIpModel.Status == 1 {
				log.WithFields(map[string]interface{}{
					"ip":     info.Ip,
					"status": honeyIpModel.Status,
				}).Warn("IP is already deploying") // IP正在部署中
				response.FailWithMsg(fmt.Sprintf("%s 正在部署中", info.Ip), c)
				return
			}
			// 状态2：IP已完成部署
			if honeyIpModel.Status == 2 {
				log.WithFields(map[string]interface{}{
					"ip":     info.Ip,
					"status": honeyIpModel.Status,
				}).Warn("IP is already a honey IP") // IP已经是诱捕ip
				response.FailWithMsg(fmt.Sprintf("%s 是诱捕ip", info.Ip), c)
				return
			}
		}

		// 组装当前IP的端口转发信息（用于MQ消息下发）
		var portList []mq_service.PortInfo
		for _, port := range hostTemplateModel.PortList {
			// 校验端口关联的虚拟服务是否存在
			service, ok1 := serviceMap[port.ServiceID]
			if !ok1 {
				log.WithFields(map[string]interface{}{
					"template_id": hostTemplateModel.ID,
					"service_id":  port.ServiceID,
				}).Warn("service not found for template port") // 模板端口关联的虚拟服务不存在
				response.FailWithMsg(
					fmt.Sprintf("主机模板%s %d 虚拟服务不存在",
						hostTemplateModel.Title, port.ServiceID), c)
				return
			}
			// 组装MQ端口信息
			portInfo := mq_service.PortInfo{
				IP:       info.Ip,
				Port:     port.Port,
				DestIP:   service.IP,
				DestPort: service.Port,
			}
			portList = append(portList, portInfo)
			// 组装待入库的诱捕端口记录
			createHoneyPortList = append(createHoneyPortList, models.HoneyPortModel{
				NodeID:    model.NodeID,
				NetID:     cr.NetID,
				ServiceID: port.ServiceID,
				IP:        info.Ip,
				Port:      port.Port,
				DstIP:     service.IP,
				DstPort:   service.Port,
				Status:    1, // 状态1：端口转发创建中
			})
		}
		// 组装当前IP的MQ部署信息
		batchDeployData.IPList = append(batchDeployData.IPList, mq_service.DeployIp{
			Ip:       info.Ip,
			Mask:     model.Mask,
			PortList: portList,
		})
		// 组装待入库的诱捕IP记录
		createHoneyIpList = append(createHoneyIpList, models.HoneyIpModel{
			NodeID:         model.NodeID,
			NetID:          cr.NetID,
			IP:             info.Ip,
			HostTemplateID: info.HostTemplateID,
			Status:         1, // 状态1：IP部署中
		})
	}

	if err := net_lock.Lock(cr.NetID); err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("failed to acquire network lock") // 锁定子网失败
		response.FailWithMsg("当前子网正在部署中", c)
		return
	}

	// 5. 数据库事务处理：入库诱捕IP/端口、初始化部署进度、下发MQ部署消息
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// Create honey IP records
		if err := tx.Create(&createHoneyIpList).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"error": err,
			}).Error("failed to create honey IPs") // 新增诱捕IP失败
			return errors.New("批量部署失败")
		}
		log.WithFields(map[string]interface{}{
			"created_ips": len(createHoneyIpList),
		}).Info("honey IPs created") // 诱捕IP创建

		// 入库诱捕端口记录（补充HoneyIpID关联）
		if len(createHoneyPortList) > 0 {
			// 构建IP到诱捕ipID的映射
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
				}).Error("failed to create honey ports") // 新增诱捕端口失败
				return errors.New("批量部署失败")
			}
			log.WithFields(map[string]interface{}{
				"created_ports": len(createPortList),
			}).Info("honey ports created") // 新增诱捕端口成功
		}

		// 初始化子网部署进度信息（存入Redis）
		if err := net_progress.Set(cr.NetID, net_progress.NetDeployInfo{
			Type:     1,                   // 部署类型：1表示批量部署
			AllCount: int64(len(cr.List)), // 总部署IP数量
		}); err != nil {
			log.WithFields(map[string]interface{}{
				"net_id": cr.NetID,
				"error":  err,
			}).Error("failed to set deployment progress") // 设置子网部署进度失败
			return errors.New("设置操作进度失败")
		}

		// 下发批量部署MQ消息（单次批量部署仅下发一条消息）
		if err := mq_service.SendBatchDeployMsg(node.Uid, batchDeployData); err != nil {
			log.WithFields(map[string]interface{}{
				"node_uid": node.Uid,
				"error":    err,
			}).Error("failed to send deployment message") // 发送部署消息失败
			return errors.New("部署消息下发失败")
		}
		return nil
	})

	// 事务执行失败处理
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("deployment transaction failed") // 部署事务执行失败
		net_lock.UnLock(cr.NetID)
		response.FailWithError(err, c)
		return
	}

	log.WithFields(map[string]interface{}{
		"net_id":       cr.NetID,
		"deployed_ips": len(createHoneyIpList),
	}).Info("batch deployment initiated successfully") // 批量部署成功启动

	mq_service.SendWsMsg(mq_service.WsMsgType{
		Type:   1,
		NetID:  cr.NetID,
		NodeID: node.ID,
	})
	response.OkWithMsg("批量部署成功，正在部署中", c)
}
