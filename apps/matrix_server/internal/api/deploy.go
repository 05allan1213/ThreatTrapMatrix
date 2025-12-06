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
	"matrix_server/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
	// 校验待部署IP列表是否为空
	if len(cr.List) == 0 {
		response.FailWithMsg("需要选择一个ip进行部署", c)
		return
	}

	// 查询子网信息并预加载关联的节点信息
	var model models.NetModel
	err := global.DB.Preload("NodeModel").Take(&model, cr.NetID).Error
	if err != nil {
		response.FailWithMsg("子网不存在", c)
		return
	}
	// 1. 校验节点在线状态（状态1表示在线）
	node := model.NodeModel
	if node.Status != 1 {
		response.FailWithMsg("节点离线", c)
		return
	}

	// 提取请求中所有非空的主机模板ID
	var hostTemplateIDList []uint
	for _, info := range cr.List {
		if info.HostTemplateID != 0 {
			hostTemplateIDList = append(hostTemplateIDList, info.HostTemplateID)
		}
	}
	// 查询主机模板列表并构建模板ID到模板的映射
	var hostTemplateList []models.HostTemplateModel
	global.DB.Find(&hostTemplateList, "id in ?", hostTemplateIDList)
	var hostTemplateMap = map[uint]models.HostTemplateModel{}

	// 提取主机模板关联的服务ID列表
	var serviceIDList []uint
	for _, templateModel := range hostTemplateList {
		hostTemplateMap[templateModel.ID] = templateModel
		for _, port := range templateModel.PortList {
			serviceIDList = append(serviceIDList, port.ServiceID)
		}
	}
	// 查询服务列表并构建服务ID到服务的映射
	var serviceList []models.ServiceModel
	global.DB.Find(&serviceList, "id in ?", serviceIDList)
	var serviceMap = map[uint]models.ServiceModel{}
	for _, serviceModel := range serviceList {
		serviceMap[serviceModel.ID] = serviceModel
	}

	// 构建子网下资产IP的映射（用于校验IP唯一性）
	var hostMap = map[string]bool{}
	var assetsList []models.HostModel
	global.DB.Find(&assetsList, "net_id = ?", cr.NetID)
	for _, hostModel := range assetsList {
		hostMap[hostModel.IP] = true
	}

	// 构建子网下诱捕IP的映射（用于校验IP部署状态）
	var honeIpMap = map[string]models.HoneyIpModel{}
	var honeyIpList []models.HoneyIpModel
	global.DB.Find(&honeyIpList, "net_id = ?", cr.NetID)
	for _, honeyModel := range honeyIpList {
		honeIpMap[honeyModel.IP] = honeyModel
	}

	// 获取上下文日志实例及日志ID
	log := middleware.GetLog(c)
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
			response.FailWithMsg(fmt.Sprintf("%d 主机模板不存在", info.HostTemplateID), c)
			return
		}
		// 3. 校验IP是否为资产IP/已部署诱捕IP
		if hostMap[info.Ip] {
			response.FailWithMsg(fmt.Sprintf("%s 是资产ip", info.Ip), c)
			return
		}
		honeyIpModel, ok := honeIpMap[info.Ip]
		if ok {
			// 状态1：IP正在部署中
			if honeyIpModel.Status == 1 {
				response.FailWithMsg(fmt.Sprintf("%s 正在部署中", info.Ip), c)
				return
			}
			// 状态2：IP已完成部署
			if honeyIpModel.Status == 2 {
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
				response.FailWithMsg(
					fmt.Sprintf("主机模板%s %d 虚拟服务不存在",
						hostTemplateModel.Title, port.ServiceID), c)
				return
			}
			// 组装MQ端口信息
			portList = append(portList, mq_service.PortInfo{
				IP:       info.Ip,
				Port:     port.Port,
				DestIP:   service.IP,
				DestPort: service.Port,
			})
			// 组装待入库的诱捕端口记录
			createHoneyPortList = append(createHoneyPortList, models.HoneyPortModel{
				NodeID: model.NodeID,
				NetID:  cr.NetID,
				// HoneyIpID：后续通过IP映射补充
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

	err = net_lock.Lock(cr.NetID)
	if err != nil {
		response.FailWithMsg("当前子网正在部署中", c)
		return
	}

	// 4. 数据库事务处理：入库诱捕IP/端口、初始化部署进度、下发MQ部署消息
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		// 入库诱捕IP记录（状态为部署中）
		var honeyIpToIDMap = map[string]uint{}
		err = global.DB.Create(&createHoneyIpList).Error
		if err != nil {
			return errors.New("批量部署失败")
		}
		logrus.Infof("批量部署%d诱捕ip", len(createHoneyIpList))

		// 入库诱捕端口记录（补充HoneyIpID关联）
		if len(createHoneyPortList) > 0 {
			// 构建IP到诱捕ipID的映射
			for _, ipModel := range createHoneyIpList {
				honeyIpToIDMap[ipModel.IP] = ipModel.ID
			}
			var createPortList []models.HoneyPortModel
			for _, portModel := range createHoneyPortList {
				portModel.HoneyIpID = honeyIpToIDMap[portModel.IP]
				createPortList = append(createPortList, portModel)
			}
			err = global.DB.Create(&createPortList).Error
			if err != nil {
				return errors.New("批量部署失败")
			}
			logrus.Infof("批量部署%d诱捕转发", len(createHoneyPortList))
		}

		// 初始化子网部署进度信息（存入Redis）
		err = net_progress.Set(cr.NetID, net_progress.NetDeployInfo{
			Type:     1,                   // 部署类型：1表示批量部署
			AllCount: int64(len(cr.List)), // 总部署IP数量
		})
		if err != nil {
			logrus.Errorf("设置操作进度失败 %s", err)
			return errors.New("设置操作进度失败")
		}

		// 下发批量部署MQ消息（单次批量部署仅下发一条消息）
		err = mq_service.SendBatchDeployMsg(node.Uid, batchDeployData)
		if err != nil {
			return errors.New("部署消息下发失败")
		}
		return nil
	})

	// 事务执行失败处理
	if err != nil {
		logrus.Errorf("部署失败 %s", err)
		response.FailWithError(err, c)
		net_lock.UnLock(cr.NetID)
		return
	}

	// 优化点：若IP数量过多，需拆分MQ消息分批下发
	response.OkWithMsg("批量部署成功，正在部署中", c)
	return
}
