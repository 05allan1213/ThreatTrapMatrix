package api

// File: matrix_server/api/update_deploy.go
// Description: 实现诱捕IP部署配置的更新API接口

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

// UpdateDeployView 批量更新已部署蜜罐IP的部署配置接口处理函数
func (Api) UpdateDeployView(c *gin.Context) {
	// 绑定并解析前端提交的批量更新部署请求参数
	cr := middleware.GetBind[DeployRequest](c)
	// 获取请求关联的日志实例（含traceID）
	log := middleware.GetLog(c)

	// 记录更新部署请求接收日志，包含子网ID和请求IP数量
	log.WithFields(map[string]interface{}{
		"net_id":        cr.NetID,
		"requested_ips": len(cr.List),
	}).Info("batch update deployment request received")

	// 校验待更新IP列表非空
	if len(cr.List) == 0 {
		log.Warn("no IPs selected for update")
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

	// 检查节点在线状态（状态1为在线，仅在线节点支持更新部署）
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

	// 收集待更新IP关联的唯一主机模板ID（去重，减少数据库查询次数）
	var hostTemplateIDList []uint
	for _, info := range cr.List {
		if info.HostTemplateID != nil {
			// 过滤空模板ID，且仅添加未存在的模板ID
			if (*info.HostTemplateID) != 0 && !utils.InList(hostTemplateIDList, *info.HostTemplateID) {
				hostTemplateIDList = append(hostTemplateIDList, *info.HostTemplateID)
			}
		}
	}

	// 加载主机模板（仅当有模板ID时）
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

	// 构建主机模板ID到模板实例的映射（便于快速查询），同时收集模板关联的服务ID
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

	// 加载子网下已部署完成的蜜罐IP（状态2）并预加载关联的端口列表，用于更新校验
	var honeyIpList []models.HoneyIpModel
	if err := global.DB.Preload("PortList").Find(
		&honeyIpList,
		"net_id = ? and status = ?",
		cr.NetID,
		2, // 2: 已部署完成状态
	).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Error("failed to load existing honey IPs")
		response.FailWithMsg("查询诱捕IP信息失败", c)
		return
	}

	// 构建蜜罐IP映射（快速判断IP是否已部署）及IP到旧模板ID的映射（用于模板变更校验）
	honeIpMap := make(map[string]models.HoneyIpModel)
	oldHoneyIpToHostTemplateMap := make(map[string]uint)
	for _, honeyModel := range honeyIpList {
		honeIpMap[honeyModel.IP] = honeyModel
		if honeyModel.HostTemplateID != nil {
			oldHoneyIpToHostTemplateMap[honeyModel.IP] = *honeyModel.HostTemplateID
		}
	}

	// 获取日志追踪ID，用于关联更新部署全流程日志
	logID := log.Data["logID"].(string)
	// 构建MQ批量更新部署消息结构体
	data := mq_service.BatchUpdateDeployRequest{
		NetID: cr.NetID,
		LogID: logID,
	}

	// 待创建的新蜜罐端口记录列表
	var createPortList []models.HoneyPortModel
	// 待删除的旧蜜罐端口记录列表
	var deletePortList []models.HoneyPortModel
	// 待更新的蜜罐IP记录列表
	var updateHoneyIpModelList []*models.HoneyIpModel

	// 遍历待更新IP列表，完成IP有效性校验并构建更新数据
	for _, info := range cr.List {
		// 校验IP是否为已部署完成的蜜罐IP
		honeyIPModel, ok := honeIpMap[info.Ip]
		if !ok {
			log.WithFields(map[string]interface{}{
				"ip": info.Ip,
			}).Warn("IP not deployed or not running")
			response.FailWithMsg(fmt.Sprintf("%s 此ip未部署", info.Ip), c)
			return
		}

		// 处理关联主机模板的更新逻辑
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

			// 校验模板是否发生变更，未变更则跳过当前IP的端口处理
			oldTemplateID := oldHoneyIpToHostTemplateMap[info.Ip]
			if oldTemplateID == *info.HostTemplateID {
				log.WithFields(map[string]interface{}{
					"ip":          info.Ip,
					"template_id": info.HostTemplateID,
					"unchanged":   true,
				}).Debug("template not changed, skipping")
				continue
			}
			log.WithFields(map[string]interface{}{
				"ip":              info.Ip,
				"old_template_id": oldTemplateID,
				"new_template_id": info.HostTemplateID,
			}).Info("template changed, preparing update")

			// 遍历新模板关联的端口，构建MQ端口更新信息及待创建的端口记录
			for _, port := range hostTemplateModel.PortList {
				// 校验模板关联的服务是否存在
				service, ok1 := serviceMap[port.ServiceID]
				if !ok1 {
					log.WithFields(map[string]interface{}{
						"template_id": info.HostTemplateID,
						"service_id":  port.ServiceID,
					}).Warn("service not found for template port")
					response.FailWithMsg(
						fmt.Sprintf("主机模板%s %d 虚拟服务不存在",
							hostTemplateModel.Title, port.ServiceID), c)
					return
				}

				// 构建MQ端口更新信息
				data.PortList = append(data.PortList, mq_service.PortInfo{
					IP:       info.Ip,
					Port:     port.Port,
					DestIP:   service.IP,
					DestPort: service.Port,
				})

				// 构建待创建的新蜜罐端口记录
				createPortList = append(createPortList, models.HoneyPortModel{
					NodeID:    model.NodeID,
					NetID:     cr.NetID,
					HoneyIpID: honeyIPModel.ID,
					ServiceID: port.ServiceID,
					IP:        info.Ip,
					Port:      port.Port,
					DstIP:     service.IP,
					DstPort:   service.Port,
					Status:    1, // 状态1：部署中
				})
			}
		} else {
			// 前端未传主机模板ID（清空模板关联），暂未处理具体逻辑
		}

		// 将当前IP添加到MQ更新的IP列表中
		data.IpList = append(data.IpList, info.Ip)

		// 收集当前IP关联的旧端口，待批量删除
		for _, portModel := range honeyIPModel.PortList {
			deletePortList = append(deletePortList, portModel)
		}

		// 更新蜜罐IP的模板关联信息，清空端口列表
		honeyIPModel.HostTemplateID = info.HostTemplateID
		honeyIPModel.PortList = []models.HoneyPortModel{}
		updateHoneyIpModelList = append(updateHoneyIpModelList, &honeyIPModel)
	}

	// 记录更新数据准备完成日志，包含待更新IP数和端口数
	log.WithFields(map[string]interface{}{
		"update_ips":   len(data.IpList),
		"update_ports": len(data.PortList),
	}).Info("prepared update data")

	// 获取子网分布式锁，防止同子网并发更新部署
	if err := net_lock.Lock(cr.NetID); err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("failed to acquire network lock")
		response.FailWithMsg("当前子网正在部署中", c)
		return
	}

	// 执行数据库事务：保证删除旧端口、更新IP、创建新端口等操作的原子性
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 批量删除旧蜜罐端口记录
		if len(deletePortList) > 0 {
			if err := tx.Delete(&deletePortList).Error; err != nil {
				log.WithFields(map[string]interface{}{
					"count": len(deletePortList), // 修复原代码count参数错误（原传deletePortList）
					"error": err,
				}).Error("failed to delete old ports")
				return errors.New("删除端口记录失败")
			}
			log.WithFields(map[string]interface{}{
				"deleted_ports": len(deletePortList),
			}).Info("deleted old ports")
		}

		// 批量更新蜜罐IP记录（模板关联信息）
		if len(updateHoneyIpModelList) > 0 {
			for _, ipModel := range updateHoneyIpModelList {
				if err := tx.Save(ipModel).Error; err != nil {
					log.WithFields(map[string]interface{}{
						"ip":    ipModel.IP,
						"error": err,
					}).Error("failed to update honey IP")
					return errors.New("更新诱捕IP记录失败")
				}
			}
			log.WithFields(map[string]interface{}{
				"updated_ips": len(updateHoneyIpModelList),
			}).Info("updated honey IPs")
		}

		// 批量创建新蜜罐端口记录
		if len(createPortList) > 0 {
			if err := tx.Create(&createPortList).Error; err != nil {
				log.WithFields(map[string]interface{}{
					"count": len(createPortList), // 修复原代码count参数错误（原传createPortList）
					"error": err,
				}).Error("failed to create new ports")
				return errors.New("创建端口记录失败")
			}
			log.WithFields(map[string]interface{}{
				"created_ports": len(createPortList),
			}).Info("created new ports")
		}

		// 设置子网更新部署进度（类型2：更新部署，总数量为待更新IP数）
		if err := net_progress.Set(cr.NetID, net_progress.NetDeployInfo{
			Type:     2, // 2: 更新部署
			AllCount: int64(len(data.IpList)),
		}); err != nil {
			log.WithFields(map[string]interface{}{
				"net_id": cr.NetID,
				"error":  err,
			}).Error("failed to set progress tracking")
			return errors.New("设置操作进度失败")
		}

		// 下发批量更新部署MQ消息到节点
		if err := mq_service.SendBatchUpdateDeployMsg(node.Uid, data); err != nil {
			log.WithFields(map[string]interface{}{
				"node_uid": node.Uid,
				"error":    err,
			}).Error("failed to send update message")
			return errors.New("更新部署消息下发失败")
		}

		return nil
	})

	// 处理事务执行结果
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("update transaction failed")
		net_lock.UnLock(cr.NetID) // 事务失败释放分布式锁，避免死锁
		response.FailWithError(err, c)
		return
	}

	// 记录更新部署启动成功日志
	log.WithFields(map[string]interface{}{
		"net_id":        cr.NetID,
		"updated_ips":   len(data.IpList),
		"updated_ports": len(data.PortList),
	}).Info("batch update deployment initiated successfully")

	// 推送WebSocket更新部署通知（类型1：部署/更新）
	mq_service.SendWsMsg(mq_service.WsMsgType{
		Type:   1,
		NetID:  cr.NetID,
		NodeID: node.ID,
	})
	response.OkWithMsg("批量更新部署成功，正在更新部署中", c)
}
