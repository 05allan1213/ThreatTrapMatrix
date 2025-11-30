package net_api

// File: honey_server/api/net_api/scan.go
// Description: 网络扫描API接口实现，处理网络扫描请求并异步接收结果更新数据库

import (
	"context"
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/utils/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ScanView 处理异步网络扫描请求的视图函数，立即返回任务启动结果并后台处理扫描数据
func (NetApi) ScanView(c *gin.Context) {
	// 获取请求绑定的ID参数（网络ID）
	cr := middleware.GetBind[models.IDRequest](c)

	var model models.NetModel
	// 查询网络模型并预加载关联的节点信息
	if err := global.DB.Preload("NodeModel").Take(&model, cr.Id).Error; err != nil {
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 检查节点是否处于运行状态
	if model.NodeModel.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 获取指定节点的命令通道
	cmd, ok := grpc_service.GetNodeCommand(model.NodeModel.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 生成唯一任务ID用于跟踪扫描任务
	taskID := fmt.Sprintf("netScan-%d", time.Now().UnixNano())
	// 构建网络扫描请求参数
	req := &node_rpc.CmdRequest{
		CmdType: node_rpc.CmdType_cmdNetScanType,
		TaskID:  taskID,
		NetScanInMessage: &node_rpc.NetScanInMessage{
			Network:      model.Network,            // 扫描使用的网络接口
			IpRange:      model.CanUseHoneyIPRange, // 待扫描的IP范围
			FilterIPList: []string{},               // 过滤IP列表
			NetID:        uint32(model.ID),         // 关联的网络ID
		},
	}

	// 非阻塞发送扫描请求到节点命令通道
	select {
	case cmd.ReqChan <- req:
		logrus.Debugf("Sent scan request to node %s, taskID: %s", model.NodeModel.Uid, taskID)
	default:
		response.FailWithMsg("发送命令通道繁忙", c)
		return
	}

	// 立即返回响应给客户端，告知任务已启动
	response.Ok(map[string]string{
		"task_id": taskID,
		"message": "扫描任务已启动，请稍后查询结果",
	}, "扫描任务已启动", c)

	// 启动异步协程处理扫描结果，不阻塞HTTP响应
	go func(nodeUid string, netModel models.NetModel, cmdChan *grpc_service.Command, taskID string) {
		// 为异步处理创建独立上下文，设置较长超时（适配扫描耗时）
		ctxAsync, cancelAsync := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancelAsync()

		// 收集有效扫描结果
		var netScanMsg []*node_rpc.NetScanOutMessage
	label:
		for {
			select {
			case res := <-cmdChan.ResChan:
				// 过滤当前任务的响应，避免处理其他任务数据
				if res.TaskID != taskID {
					// 非当前任务响应，放回通道供其他协程处理
					select {
					case cmdChan.ResChan <- res:
					case <-ctxAsync.Done():
						break label
					}
					continue
				}

				logrus.Debugf("Received scan response from node %s, taskID: %s", nodeUid, taskID)
				message := res.NetScanOutMessage

				// 扫描过程出现错误则终止接收
				if message.ErrMsg != "" {
					logrus.Errorf("Scan error from node %s: %s", nodeUid, message.ErrMsg)
					break label
				}

				// 扫描完成则跳出循环
				if message.End {
					break label
				}

				// 收集有效主机信息
				if message.Ip != "" {
					netScanMsg = append(netScanMsg, message)
				}

			case <-ctxAsync.Done():
				logrus.Errorf("Scan timeout for node %s, taskID: %s", nodeUid, taskID)
				return
			}
		}

		// 有效结果或未超时情况下处理扫描数据
		if len(netScanMsg) > 0 || ctxAsync.Err() == nil {
			processScanResult(netModel, netScanMsg)
		} else {
			logrus.Warnf("No scan results received for task %s", taskID)
		}

	}(model.NodeModel.Uid, model, cmd, taskID)
}

// processScanResult 处理扫描结果，对比数据库主机信息并执行增删改操作
func processScanResult(netModel models.NetModel, scanMsgs []*node_rpc.NetScanOutMessage) {
	// 查询当前网络下已存在的主机列表
	var hostList []models.HostModel
	if err := global.DB.Find(&hostList, "net_id = ?", netModel.ID).Error; err != nil {
		logrus.Errorf("Failed to get host list for net %d: %v", netModel.ID, err)
		return
	}

	// 构建数据库主机IP映射（便于快速查找）
	dbHostMap := make(map[string]models.HostModel)
	for _, host := range hostList {
		dbHostMap[host.IP] = host
	}

	// 构建扫描结果IP映射
	scanResultMap := make(map[string]*node_rpc.NetScanOutMessage)
	for _, msg := range scanMsgs {
		if msg.Ip != "" {
			scanResultMap[msg.Ip] = msg
		}
	}

	// 分类处理：新增主机、更新主机、删除主机
	var newHosts []models.HostModel     // 新增主机列表
	var deletedHostIDs []uint           // 待删除主机ID列表
	var updatedHosts []models.HostModel // 待更新主机列表

	// 处理新增和更新逻辑
	for ip, scanMsg := range scanResultMap {
		if dbHost, exists := dbHostMap[ip]; exists {
			// 主机已存在，检查MAC/厂商信息是否变化
			if dbHost.Mac != scanMsg.Mac || dbHost.Manuf != scanMsg.Manuf {
				dbHost.Mac = scanMsg.Mac
				dbHost.Manuf = scanMsg.Manuf
				updatedHosts = append(updatedHosts, dbHost)
			}
			delete(dbHostMap, ip) // 标记为已处理
		} else {
			// 主机不存在，加入新增列表
			newHosts = append(newHosts, models.HostModel{
				NodeID: netModel.NodeModel.ID,
				NetID:  netModel.ID,
				IP:     scanMsg.Ip,
				Mac:    scanMsg.Mac,
				Manuf:  scanMsg.Manuf,
			})
		}
	}

	// 剩余未处理的数据库主机即为需删除的主机
	for _, dbHost := range dbHostMap {
		deletedHostIDs = append(deletedHostIDs, dbHost.ID)
	}

	logrus.Infof("Net %d scan result: new=%d, updated=%d, deleted=%d",
		netModel.ID, len(newHosts), len(updatedHosts), len(deletedHostIDs))

	// 使用事务批量更新数据库，保证数据一致性
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 批量创建新增主机
		if len(newHosts) > 0 {
			if err := tx.Create(&newHosts).Error; err != nil {
				return fmt.Errorf("创建新主机失败: %w", err)
			}
		}

		// 逐个更新主机信息
		if len(updatedHosts) > 0 {
			for _, host := range updatedHosts {
				if err := tx.Model(&models.HostModel{}).
					Where("id = ?", host.ID).
					Updates(map[string]interface{}{
						"mac":   host.Mac,
						"manuf": host.Manuf,
					}).Error; err != nil {
					return fmt.Errorf("更新主机信息失败: %w", err)
				}
			}
		}

		// 批量删除失效主机
		if len(deletedHostIDs) > 0 {
			if err := tx.Delete(&models.HostModel{}, deletedHostIDs).Error; err != nil {
				return fmt.Errorf("删除主机失败: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		logrus.Errorf("Failed to update scan results for net %d: %v", netModel.ID, err)
	} else {
		logrus.Infof("Successfully updated scan results for net %d", netModel.ID)
	}
}
