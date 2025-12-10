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

// UpdateDeployView 子网批量更新部署接口处理函数
func (Api) UpdateDeployView(c *gin.Context) {
	// 1. 解析并绑定请求参数
	cr := middleware.GetBind[DeployRequest](c)
	log := middleware.GetLog(c)

	log.WithFields(map[string]interface{}{
		"net_id":        cr.NetID,
		"requested_ips": len(cr.List),
	}).Info("batch update deployment request received") // 收到批量更新部署请求

	// 校验请求IP列表非空
	if len(cr.List) == 0 {
		log.Warn("no IPs selected for update") // 无IP被选中
		response.FailWithMsg("需要选择一个ip进行部署", c)
		return
	}

	// 2. 校验子网是否存在（预加载关联节点信息）
	var model models.NetModel
	if err := global.DB.Preload("NodeModel").Take(&model, cr.NetID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("subnet not found") // 子网未找到
		response.FailWithMsg("子网不存在", c)
		return
	}

	// 3. 校验节点是否在线（状态1为运行中）
	node := model.NodeModel
	if node.Status != 1 {
		log.WithFields(map[string]interface{}{
			"node_id":  node.ID,
			"node_uid": node.Uid,
			"status":   node.Status,
		}).Warn("node is offline") // 节点离线
		response.FailWithMsg("节点离线", c)
		return
	}

	// 4. 收集请求中唯一的主机模板ID（去重）
	var hostTemplateIDList []uint
	for _, info := range cr.List {
		if info.HostTemplateID != 0 && !utils.InList(hostTemplateIDList, info.HostTemplateID) {
			hostTemplateIDList = append(hostTemplateIDList, info.HostTemplateID)
		}
	}

	// 5. 加载主机模板信息（关联端口配置）
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

	// 6. 构建主机模板映射，收集关联的服务ID（去重）
	hostTemplateMap := make(map[uint]models.HostTemplateModel)
	var serviceIDList []uint
	for _, templateModel := range hostTemplateList {
		hostTemplateMap[templateModel.ID] = templateModel
		for _, port := range templateModel.PortList {
			if !utils.InList(serviceIDList, port.ServiceID) {
				serviceIDList = append(serviceIDList, port.ServiceID)
			}
		}
	}

	// 7. 加载服务信息（端口转发的目标服务配置）
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

	// 8. 构建服务信息映射（便于快速查询）
	serviceMap := make(map[uint]models.ServiceModel)
	for _, serviceModel := range serviceList {
		serviceMap[serviceModel.ID] = serviceModel
	}

	// 9. 加载子网下已部署的诱捕IP（状态2为已部署，预加载关联端口）
	var honeyIpList []models.HoneyIpModel
	if err := global.DB.Preload("PortList").Find(
		&honeyIpList,
		"net_id = ? and status = ?",
		cr.NetID,
		2, // 2: 已部署状态
	).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Error("failed to load existing honey IPs") // 查询诱捕IP信息失败
		response.FailWithMsg("查询诱捕IP信息失败", c)
		return
	}

	// 10. 构建诱捕IP映射（IP→模型）和旧模板映射（IP→原模板ID）
	honeIpMap := make(map[string]models.HoneyIpModel)
	oldHoneyIpToHostTemplateMap := make(map[string]uint)
	for _, honeyModel := range honeyIpList {
		honeIpMap[honeyModel.IP] = honeyModel
		oldHoneyIpToHostTemplateMap[honeyModel.IP] = honeyModel.HostTemplateID
	}

	// 11. 构建MQ下发的更新请求数据
	logID := log.Data["logID"].(string)
	data := mq_service.BatchUpdateDeployRequest{
		NetID: cr.NetID,
		LogID: logID,
	}

	// 初始化端口操作列表：待删除旧端口、待创建新端口、待更新诱捕IP
	var createPortList []models.HoneyPortModel
	var deletePortList []models.HoneyPortModel
	var updateHoneyIpModelList []*models.HoneyIpModel

	// 12. 遍历请求IP列表，校验并准备更新数据
	for _, info := range cr.List {
		// 校验主机模板存在
		hostTemplateModel, ok := hostTemplateMap[info.HostTemplateID]
		if !ok {
			log.WithFields(map[string]interface{}{
				"template_id": info.HostTemplateID,
				"ip":          info.Ip,
			}).Warn("host template not found") // 主机模板未找到
			response.FailWithMsg(fmt.Sprintf("%d 主机模板不存在", info.HostTemplateID), c)
			return
		}

		// 校验诱捕IP已部署且运行中
		honeyIPModel, ok := honeIpMap[info.Ip]
		if !ok {
			log.WithFields(map[string]interface{}{
				"ip": info.Ip,
			}).Warn("IP not deployed or not running") // IP未部署或未运行
			response.FailWithMsg(fmt.Sprintf("%s 此ip未部署", info.Ip), c)
			return
		}

		// 模板未变更则跳过当前IP
		oldTemplateID := oldHoneyIpToHostTemplateMap[info.Ip]
		if oldTemplateID == info.HostTemplateID {
			log.WithFields(map[string]interface{}{
				"ip":          info.Ip,
				"template_id": info.HostTemplateID,
				"unchanged":   true,
			}).Debug("template not changed, skipping") // 模板未变更，跳过当前IP
			continue
		}

		log.WithFields(map[string]interface{}{
			"ip":              info.Ip,
			"old_template_id": oldTemplateID,
			"new_template_id": info.HostTemplateID,
		}).Info("template changed, preparing update") // 模板已变更，准备更新

		// 加入待更新IP列表
		data.IpList = append(data.IpList, info.Ip)

		// 收集待删除的旧端口（原模板关联的端口）
		for _, portModel := range honeyIPModel.PortList {
			deletePortList = append(deletePortList, portModel)
		}

		// 更新诱捕IP的模板ID，清空旧端口列表
		honeyIPModel.HostTemplateID = info.HostTemplateID
		honeyIPModel.PortList = []models.HoneyPortModel{}
		updateHoneyIpModelList = append(updateHoneyIpModelList, &honeyIPModel)

		// 收集待创建的新端口（新模板关联的端口）
		for _, port := range hostTemplateModel.PortList {
			// 校验服务存在
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

			// 加入MQ下发的端口列表
			data.PortList = append(data.PortList, mq_service.PortInfo{
				IP:       info.Ip,
				Port:     port.Port,
				DestIP:   service.IP,
				DestPort: service.Port,
			})

			// 加入待创建的端口记录列表
			createPortList = append(createPortList, models.HoneyPortModel{
				NodeID:    model.NodeID,
				NetID:     cr.NetID,
				HoneyIpID: honeyIPModel.ID,
				ServiceID: port.ServiceID,
				IP:        info.Ip,
				Port:      port.Port,
				DstIP:     service.IP,
				DstPort:   service.Port,
				Status:    1, // 1: 端口正常状态
			})
		}
	}

	// 无端口更新则返回
	if len(data.PortList) == 0 {
		log.Info("no ports to update, all templates unchanged") // 无端口更新，所有模板未变更
		response.FailWithMsg("没有需要部署的操作", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"update_ips":   len(data.IpList),
		"update_ports": len(data.PortList),
	}).Info("prepared update data") // 准备更新数据

	// 13. 获取子网分布式锁（防止并发部署）
	if err := net_lock.Lock(cr.NetID); err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.NetID,
			"error":  err,
		}).Warn("failed to acquire network lock") // 获取子网分布式锁失败
		response.FailWithMsg("当前子网正在部署中", c)
		return
	}

	// 14. 事务执行数据更新（保证原子性）
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 步骤1：删除旧端口记录
		if len(deletePortList) > 0 {
			if err := tx.Delete(&deletePortList).Error; err != nil {
				log.WithFields(map[string]interface{}{
					"count": len(deletePortList),
					"error": err,
				}).Error("failed to delete old ports") // 删除旧端口记录失败
				return errors.New("删除端口记录失败")
			}
			log.WithFields(map[string]interface{}{
				"deleted_ports": len(deletePortList),
			}).Info("deleted old ports") // 删除旧端口记录成功
		}

		// 步骤2：更新诱捕IP的模板ID
		if len(updateHoneyIpModelList) > 0 {
			for _, ipModel := range updateHoneyIpModelList {
				if err := tx.Save(ipModel).Error; err != nil {
					log.WithFields(map[string]interface{}{
						"ip":    ipModel.IP,
						"error": err,
					}).Error("failed to update honey IP") // 更新诱捕IP记录失败
					return errors.New("更新诱捕IP记录失败")
				}
			}
			log.WithFields(map[string]interface{}{
				"updated_ips": len(updateHoneyIpModelList),
			}).Info("updated honey IPs") // 更新诱捕IP记录成功
		}

		// 步骤3：创建新端口记录
		if len(createPortList) > 0 {
			if err := tx.Create(&createPortList).Error; err != nil {
				log.WithFields(map[string]interface{}{
					"count": len(createPortList),
					"error": err,
				}).Error("failed to create new ports") // 创建新端口记录失败
				return errors.New("创建端口记录失败")
			}
			log.WithFields(map[string]interface{}{
				"created_ports": len(createPortList),
			}).Info("created new ports") // 创建新端口记录成功
		}

		// 步骤4：设置更新部署进度（Type=2为更新部署）
		if err := net_progress.Set(cr.NetID, net_progress.NetDeployInfo{
			Type:     2, // 2: 更新部署
			AllCount: int64(len(data.IpList)),
		}); err != nil {
			log.WithFields(map[string]interface{}{
				"net_id": cr.NetID,
				"error":  err,
			}).Error("failed to set progress tracking") // 设置操作进度失败
			return errors.New("设置操作进度失败")
		}

		// 步骤5：下发更新部署MQ指令至节点
		if err := mq_service.SendBatchUpdateDeployMsg(node.Uid, data); err != nil {
			log.WithFields(map[string]interface{}{
				"node_uid": node.Uid,
				"error":    err,
			}).Error("failed to send update message") // 更新部署消息下发失败
			return errors.New("更新部署消息下发失败")
		}

		return nil
	})

	// 15. 事务失败处理：释放分布式锁，返回错误
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("update transaction failed") // 更新事务失败
		net_lock.UnLock(cr.NetID)             // 失败时释放锁
		response.FailWithError(err, c)
		return
	}

	log.WithFields(map[string]interface{}{
		"net_id":        cr.NetID,
		"updated_ips":   len(data.IpList),
		"updated_ports": len(data.PortList),
	}).Info("batch update deployment initiated successfully") // 批量更新部署成功，正在更新部署中

	// 16. 发送更新部署MQ消息
	mq_service.SendWsMsg(mq_service.WsMsgType{
		Type:   1,
		NetID:  cr.NetID,
		NodeID: node.ID,
	})
	// 17. 返回成功响应
	response.OkWithMsg("批量更新部署成功，正在更新部署中", c)
}
