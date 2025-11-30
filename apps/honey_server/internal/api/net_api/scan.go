package net_api

// File: honey_server/api/net_api/scan.go
// Description: 网络扫描API接口，实现扫描状态互斥与诱捕IP过滤，异步处理扫描结果并更新数据库

import (
	"context"
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/utils/response"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var mutex sync.Mutex        // 全局互斥锁，控制子网扫描并发执行
var netProgressMap sync.Map // 存储各子网扫描进度，key为网络ID，value为进度百分比

// ScanView 带进度跟踪的网络扫描请求处理函数，支持扫描状态控制与进度实时更新
func (NetApi) ScanView(c *gin.Context) {
	// 获取请求绑定的网络ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	var model models.NetModel
	// 查询网络信息并预加载关联节点数据
	if err := global.DB.Preload("NodeModel").Take(&model, cr.Id).Error; err != nil {
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 检查节点运行状态
	if model.NodeModel.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 计算可用IP范围
	if model.CanUseHoneyIPRange == "" {
		response.FailWithMsg("网络可使用ip范围为空", c)
		return
	}

	// 获取节点命令通道，用于与节点通信
	cmd, ok := grpc_service.GetNodeCommand(model.NodeModel.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 查询当前网络下的诱捕IP列表，用于扫描过滤
	var filterIPList []string
	global.DB.Model(models.HoneyIpModel{}).Where("net_id = ?", cr.Id).Select("ip").Scan(&filterIPList)
	fmt.Println("过滤的ip列表", filterIPList)

	// 生成唯一扫描任务ID
	taskID := fmt.Sprintf("netScan-%d", time.Now().UnixNano())
	// 构建扫描请求参数
	req := &node_rpc.CmdRequest{
		CmdType: node_rpc.CmdType_cmdNetScanType,
		TaskID:  taskID,
		NetScanInMessage: &node_rpc.NetScanInMessage{
			Network:      model.Network,            // 扫描使用的网络接口
			IpRange:      model.CanUseHoneyIPRange, // 待扫描IP范围
			FilterIPList: filterIPList,             // 过滤的诱捕IP列表
			NetID:        uint32(model.ID),         // 关联网络ID
		},
	}

	// 互斥锁控制，避免同一子网并发扫描
	mutex.Lock()
	if model.ScanStatus == 2 {
		response.FailWithMsg("当前子网正在扫描中", c)
		mutex.Unlock()
		return
	}

	// 更新网络扫描状态为"扫描中"
	global.DB.Model(&model).Update("scan_status", 2)
	mutex.Unlock()

	// 非阻塞发送扫描请求到节点
	select {
	case cmd.ReqChan <- req:
		logrus.Debugf("Sent scan request to node %s, taskID: %s", model.NodeModel.Uid, taskID)
	default:
		response.FailWithMsg("发送命令通道繁忙", c)
		return
	}

	// 立即返回响应，告知客户端任务启动成功
	response.Ok(map[string]string{
		"task_id": taskID,
		"message": "扫描任务已启动，请稍后查询结果",
	}, "扫描任务已启动", c)

	// 异步协程处理扫描结果与进度更新
	go func(nodeUid string, netModel models.NetModel, cmdChan *grpc_service.Command, taskID string) {
		// 设置异步处理超时时间（5分钟）
		ctxAsync, cancelAsync := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancelAsync()

		// 收集扫描结果
		var netScanMsg []*node_rpc.NetScanOutMessage
	label:
		for {
			select {
			case res := <-cmdChan.ResChan:
				// 过滤当前任务的响应数据
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

				// 扫描出错时终止处理
				if message.ErrMsg != "" {
					logrus.Errorf("Scan error from node %s: %s", nodeUid, message.ErrMsg)
					break label
				}

				// 扫描完成时跳出循环
				if message.End {
					break label
				}

				// 收集有效主机信息并更新扫描进度
				if message.Ip != "" {
					netScanMsg = append(netScanMsg, message)
					netProgressMap.Store(uint(message.NetID), float64(message.Progress)) // 存储当前进度
					fmt.Printf("网络扫描 %s %s %s %.2f\n", message.Ip, message.Mac, message.Manuf, message.Progress)
				}

			case <-ctxAsync.Done():
				logrus.Errorf("Scan timeout for node %s, taskID: %s", nodeUid, taskID)
				return
			}
		}

		// 处理并持久化扫描结果
		if len(netScanMsg) > 0 || ctxAsync.Err() == nil {
			processScanResult(netModel, netScanMsg)
		} else {
			logrus.Warnf("No scan results received for task %s", taskID)
		}

	}(model.NodeModel.Uid, model, cmd, taskID)
}

// processScanResult 处理扫描结果，更新主机信息并重置扫描状态
func processScanResult(netModel models.NetModel, scanMsgs []*node_rpc.NetScanOutMessage) {
	// 最终更新扫描状态为完成，并清理进度缓存
	defer func() {
		global.DB.Model(&netModel).Updates(map[string]any{
			"scan_progress": 100, // 进度置为100%
			"scan_status":   1,   // 状态置为已完成
		})
		netProgressMap.Delete(netModel.ID) // 移除进度缓存
	}()

	// 查询当前网络下已存在的主机列表
	var hostList []models.HostModel
	if err := global.DB.Find(&hostList, "net_id = ?", netModel.ID).Error; err != nil {
		logrus.Errorf("Failed to get host list for net %d: %v", netModel.ID, err)
		return
	}

	// 构建数据库主机IP映射表，用于快速比对
	dbHostMap := make(map[string]models.HostModel)
	for _, host := range hostList {
		dbHostMap[host.IP] = host
	}

	// 构建扫描结果IP映射表
	scanResultMap := make(map[string]*node_rpc.NetScanOutMessage)
	for _, msg := range scanMsgs {
		if msg.Ip != "" {
			scanResultMap[msg.Ip] = msg
		}
	}

	// 分类存储需新增、更新、删除的主机信息
	var newHosts []models.HostModel
	var deletedHostIDs []uint
	var updatedHosts []models.HostModel

	// 处理新增与更新逻辑
	for ip, scanMsg := range scanResultMap {
		if dbHost, exists := dbHostMap[ip]; exists {
			// 主机已存在，检查信息是否变化
			if dbHost.Mac != scanMsg.Mac || dbHost.Manuf != scanMsg.Manuf {
				dbHost.Mac = scanMsg.Mac
				dbHost.Manuf = scanMsg.Manuf
				updatedHosts = append(updatedHosts, dbHost)
			}
			delete(dbHostMap, ip) // 标记为已处理
		} else {
			// 新增主机记录
			newHosts = append(newHosts, models.HostModel{
				NodeID: netModel.NodeModel.ID,
				NetID:  netModel.ID,
				IP:     scanMsg.Ip,
				Mac:    scanMsg.Mac,
				Manuf:  scanMsg.Manuf,
			})
		}
	}

	// 剩余未处理的主机即为需删除的记录
	for _, dbHost := range dbHostMap {
		deletedHostIDs = append(deletedHostIDs, dbHost.ID)
	}

	logrus.Infof("Net %d scan result: new=%d, updated=%d, deleted=%d",
		netModel.ID, len(newHosts), len(updatedHosts), len(deletedHostIDs))

	// 事务批量更新数据库，保证数据一致性
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 批量创建新主机
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
