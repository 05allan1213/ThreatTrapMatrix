package api

// File: matrix_server/api/deploy.go
// Description: 实现诱捕IP批量部署API接口

import (
	"context"
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
	// 绑定并解析前端传入的部署请求参数
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

	// 1. 校验子网关联的节点是否在线
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

	// 查询主机模板列表，并构建模板ID到模板信息的映射表
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

	// 查询服务列表，并构建服务ID到服务信息的映射表
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

	// 获取请求上下文的日志信息，提取日志ID用于部署追踪
	log := middleware.GetLog(c)
	logID := log.Data["logID"].(string)

	// 初始化MQ批量部署指令数据结构体
	var batchDeployData = mq_service.BatchDeployRequest{
		NetID:   cr.NetID,
		LogID:   logID,
		Network: model.Network,
		TanIp:   model.IP,
	}

	// 初始化待入库的诱捕IP列表、待入库的诱捕端口列表
	var createHoneyIpList []models.HoneyIpModel
	var createHoneyPortList []models.HoneyPortModel

	// 构建Redis中部署中IP的缓存key，用于校验IP是否正在部署
	key := fmt.Sprintf("deploy_create_%d", cr.NetID)
	// 从Redis获取当前子网下所有部署中的IP状态
	maps := global.Redis.HGetAll(context.Background(), key).Val()
	var creatingMap = map[string]bool{}
	for k, s2 := range maps {
		if s2 == "1" {
			creatingMap[k] = true // 标记该IP处于部署中状态
		}
	}

	// 2. 逐行校验待部署IP及关联主机模板的合法性
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

		// 4. 校验IP合法性：不能是正在部署中/删除中的IP
		if creatingMap[info.Ip] {
			response.FailWithMsg(fmt.Sprintf("%s 正在部署中", info.Ip), c)
			return
		}

		// 解析主机模板的端口列表，构建MQ下发的端口转发配置
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

			// 构建MQ下发的端口转发配置信息
			portList = append(portList, mq_service.PortInfo{
				IP:       info.Ip,
				Port:     port.Port,
				DestIP:   service.IP,
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
				Status:    1, // 状态标记为创建中
			})
		}

		// 组装MQ批量部署指令中的IP配置数据
		batchDeployData.IPList = append(batchDeployData.IPList, mq_service.DeployIp{
			Ip:       info.Ip,
			Mask:     model.Mask,
			PortList: portList,
		})

		// 构建待入库的诱捕IP数据
		createHoneyIpList = append(createHoneyIpList, models.HoneyIpModel{
			NodeID: model.NodeID,
			NetID:  cr.NetID,
			IP:     info.Ip,
			Status: 1, // 状态标记为创建中
		})

		// 将当前IP标记为部署中状态，写入Redis缓存
		global.Redis.HSet(context.Background(), key, info.Ip, true)
	}

	// 5. 分布式锁控制：防止同一子网并发部署
	// 创建redsync的Redis连接池
	pool := goredis.NewPool(global.Redis)
	// 初始化redsync实例
	rs := redsync.New(pool)
	// 构建子网部署锁的key
	mutexname := fmt.Sprintf("deploy_create_lock_%d", cr.NetID)
	// 创建基于该key的互斥锁，配置过期时间、重试次数和重试间隔
	mutex := rs.NewMutex(mutexname,
		redsync.WithExpiry(20*time.Minute),           // 锁过期时间20分钟，防止死锁
		redsync.WithTries(1),                         // 仅重试1次
		redsync.WithRetryDelay(500*time.Millisecond), // 重试间隔500毫秒
	)

	// 尝试获取分布式锁
	if err1 := mutex.Lock(); err1 != nil {
		response.FailWithMsg("当前子网正在部署中", c)
		global.Redis.Del(context.Background(), key)
		return
	}

	// 数据库事务：批量创建诱捕IP/端口数据，并下发MQ部署指令
	// 事务内任一操作失败则全部回滚
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		// 批量创建诱捕IP数据，状态设为创建中
		err = global.DB.Create(&createHoneyIpList).Error
		if err != nil {
			return errors.New("批量部署失败")
		}
		logrus.Infof("批量部署%d诱捕ip", len(createHoneyIpList))

		// 批量创建诱捕端口转发数据（如有）
		if len(createHoneyPortList) > 0 {
			err = global.DB.Create(&createHoneyPortList).Error
			if err != nil {
				return errors.New("批量部署失败")
			}
			logrus.Infof("批量部署%d诱捕转发", len(createHoneyPortList))
		}

		// 下发批量部署指令到MQ队列，指定目标节点UID
		err = mq_service.SendBatchDeployMsg(node.Uid, batchDeployData)
		if err != nil {
			return errors.New("部署消息下发失败")
		}
		return nil
	})

	// 处理事务执行失败的情况
	if err != nil {
		logrus.Errorf("部署失败 %s", err)
		response.FailWithError(err, c)
		global.Redis.Del(context.Background(), key)
		return
	}

	// 待优化点：若待部署IP数量过多，需拆分MQ消息分批下发

	// 返回部署指令下发成功响应
	response.OkWithMsg("批量部署成功，正在部署中", c)
	return
}
