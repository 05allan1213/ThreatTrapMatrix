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
	"matrix_server/internal/utils/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// UpdateDeployView 处理诱捕IP部署配置更新请求
func (Api) UpdateDeployView(c *gin.Context) {
	// 绑定并解析前端传入的部署更新请求参数
	cr := middleware.GetBind[DeployRequest](c)
	// 校验待更新IP列表不能为空
	if len(cr.List) == 0 {
		response.FailWithMsg("需要选择一个ip进行部署", c)
		return
	}

	// 查询指定子网信息，并预加载关联的节点信息
	var model models.NetModel
	err := global.DB.Preload("NodeModel").Take(&model, cr.NetID).Error
	if err != nil {
		response.FailWithMsg("子网不存在", c)
		return
	}
	// 1. 校验子网关联的节点是否在线（仅在线节点可执行更新操作）
	node := model.NodeModel
	if node.Status != 1 {
		response.FailWithMsg("节点离线", c)
		return
	}

	// ---------------------- 步骤1：校验更新的主机模板，构建模板/服务映射表 ----------------------
	// 提取待更新IP关联的主机模板ID列表
	var hostTemplateIDList []uint
	for _, info := range cr.List {
		if info.HostTemplateID != 0 {
			hostTemplateIDList = append(hostTemplateIDList, info.HostTemplateID)
		}
	}
	// 查询主机模板列表，构建模板ID到模板信息的映射表
	var hostTemplateList []models.HostTemplateModel
	global.DB.Find(&hostTemplateList, "id in ?", hostTemplateIDList)
	var hostTemplateMap = map[uint]models.HostTemplateModel{}

	// 提取主机模板关联的虚拟服务ID列表
	var serviceIDList []uint
	for _, templateModel := range hostTemplateList {
		hostTemplateMap[templateModel.ID] = templateModel
		for _, port := range templateModel.PortList {
			serviceIDList = append(serviceIDList, port.ServiceID)
		}
	}
	// 查询虚拟服务列表，构建服务ID到服务信息的映射表
	var serviceList []models.ServiceModel
	global.DB.Find(&serviceList, "id in ?", serviceIDList)
	var serviceMap = map[uint]models.ServiceModel{}
	for _, serviceModel := range serviceList {
		serviceMap[serviceModel.ID] = serviceModel
	}

	// ---------------------- 步骤2：查询子网下运行中的IP，构建IP/模板映射关系 ----------------------
	// honeIpMap: 运行中IP -> HoneyIpModel映射（快速查询IP部署信息）
	var honeIpMap = map[string]models.HoneyIpModel{}
	// oldHoneyIpToHostTemplateMap: 运行中IP -> 原主机模板ID映射（用于判断模板是否变更）
	var oldHoneyIpToHostTemplateMap = map[string]uint{}
	// 查询子网下状态为2（运行中）的诱捕IP，预加载关联的端口列表
	var honeyIpList []models.HoneyIpModel
	global.DB.Preload("PortList").Find(&honeyIpList, "net_id = ? and status = ?", cr.NetID, 2)
	for _, honeyModel := range honeyIpList {
		honeIpMap[honeyModel.IP] = honeyModel
		oldHoneyIpToHostTemplateMap[honeyModel.IP] = honeyModel.HostTemplateID
	}

	// 获取请求上下文的日志ID，用于关联更新操作的日志记录
	log := middleware.GetLog(c)
	logID := log.Data["logID"].(string)
	// 初始化MQ批量更新部署指令数据结构体
	data := mq_service.BatchUpdateDeployRequest{
		NetID: cr.NetID,
		LogID: logID,
	}

	// deletePortList: 待删除的旧端口记录列表（模板变更后需删除原端口）
	var deletePortList []models.HoneyPortModel
	// createPortList: 待创建的新端口记录列表（模板变更后需创建新端口）
	var createPortList []models.HoneyPortModel
	// updateHoneyIpModelList: 待更新IP记录列表（模板变更后需更新IP记录）
	var updateHoneyIpModelList []*models.HoneyIpModel

	// ---------------------- 步骤3：筛选模板变更的IP，构建端口增删列表 ----------------------
	for _, info := range cr.List {
		// 校验主机模板是否存在
		hostTemplateModel, ok := hostTemplateMap[info.HostTemplateID]
		if !ok {
			response.FailWithMsg(fmt.Sprintf("%d 主机模板不存在", info.HostTemplateID), c)
			return
		}
		// 校验IP是否为运行中的诱捕IP（未部署则拒绝更新）
		honeyIPModel, ok := honeIpMap[info.Ip]
		if !ok {
			response.FailWithMsg(fmt.Sprintf("%s 此ip未部署", info.Ip), c)
			return
		}

		// 对比原模板ID与新模板ID，无变更则跳过当前IP
		oldTemplateID := oldHoneyIpToHostTemplateMap[info.Ip]
		if oldTemplateID == info.HostTemplateID {
			continue
		}
		logrus.Infof("%s 更新了主机模板 %d=>%d", info.Ip, oldTemplateID, info.HostTemplateID)

		// 将模板变更的IP加入MQ更新列表
		data.IpList = append(data.IpList, info.Ip)

		var portList []mq_service.PortInfo
		// 收集当前IP的旧端口记录（待删除）
		for _, portModel := range honeyIPModel.PortList {
			deletePortList = append(deletePortList, portModel)
		}

		honeyIPModel.HostTemplateID = info.HostTemplateID
		honeyIPModel.PortList = []models.HoneyPortModel{}
		updateHoneyIpModelList = append(updateHoneyIpModelList, &honeyIPModel)

		// 构建新端口列表（基于新主机模板）
		for _, port := range hostTemplateModel.PortList {
			// 校验虚拟服务是否存在
			service, ok1 := serviceMap[port.ServiceID]
			if !ok1 {
				response.FailWithMsg(
					fmt.Sprintf("主机模板%s %d 虚拟服务不存在",
						hostTemplateModel.Title, port.ServiceID), c)
				return
			}
			// 构建MQ下发的端口转发配置
			portList = append(portList, mq_service.PortInfo{
				IP:       info.Ip,
				Port:     port.Port,
				DestIP:   service.IP,
				DestPort: service.Port,
			})
			// 构建待创建的端口记录（入库用）
			createPortList = append(createPortList, models.HoneyPortModel{
				NodeID:    model.NodeID,
				NetID:     cr.NetID,
				HoneyIpID: honeyIPModel.ID,
				ServiceID: port.ServiceID,
				IP:        info.Ip,
				Port:      port.Port,
				DstIP:     service.IP,
				DstPort:   service.Port,
				Status:    1, // 状态1：创建中
			})
		}

		// 将新端口配置加入MQ更新指令
		data.PortList = portList
	}

	// ---------------------- 步骤4：分布式锁控制子网并发操作 ----------------------
	// 创建redsync的Redis连接池
	pool := goredis.NewPool(global.Redis)
	// 初始化redsync实例
	rs := redsync.New(pool)
	// 构建子网更新操作锁的key（避免同一子网同时执行部署/更新操作）
	mutexname := fmt.Sprintf("deploy_action_lock_%d", cr.NetID)
	// 创建基于该key的互斥锁（配置与部署锁一致）
	mutex := rs.NewMutex(mutexname,
		redsync.WithExpiry(20*time.Minute),           // 锁过期时间20分钟，防止死锁
		redsync.WithTries(1),                         // 仅重试1次
		redsync.WithRetryDelay(500*time.Millisecond), // 重试间隔500毫秒
	)

	// 尝试获取分布式锁（获取失败说明子网正在执行其他操作）
	if err1 := mutex.Lock(); err1 != nil {
		response.FailWithMsg("当前子网正在部署中", c)
		return
	}

	// ---------------------- 步骤5：事务更新端口记录，下发MQ更新指令 ----------------------
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		// 删除旧端口记录（模板变更的IP需清理原有端口）
		if len(deletePortList) > 0 {
			err = global.DB.Delete(&deletePortList).Error
			if err != nil {
				return errors.New("删除端口记录失败")
			}
			logrus.Infof("删除端口记录 %d个", len(deletePortList))
		}

		// 更新IP记录（模板变更后需更新IP记录）
		if len(updateHoneyIpModelList) > 0 {
			for _, ipModel := range updateHoneyIpModelList {
				global.DB.Save(ipModel)
			}
		}

		// 创建新端口记录（基于新主机模板）
		if len(createPortList) > 0 {
			err = global.DB.Create(&createPortList).Error
			if err != nil {
				return errors.New("创建端口记录失败")
			}
			logrus.Infof("创建端口记录 %d 个", len(createPortList))
		}

		// 下发批量更新部署指令到MQ队列（指定目标节点UID）
		err = mq_service.SendBatchUpdateDeployMsg(node.Uid, data)
		if err != nil {
			return errors.New("更新部署消息下发失败")
		}
		return nil
	})

	// 处理事务执行失败的情况
	if err != nil {
		logrus.Errorf("部署失败 %s", err)
		response.FailWithError(err, c)
		return
	}

	// 返回更新指令下发成功响应
	response.OkWithMsg("批量更新部署成功，正在更新部署中", c)
	return
}
