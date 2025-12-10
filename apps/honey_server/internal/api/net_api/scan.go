package net_api

// File: honey_server/api/net_api/scan.go
// Description: 网络扫描API接口，实现扫描状态互斥与诱捕IP过滤，异步处理扫描结果并更新数据库

import (
	"context"
	"fmt"
	"honey_server/internal/core"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/service/mq_service"
	"honey_server/internal/utils/response"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var mutex sync.Mutex        // 全局互斥锁，控制子网扫描并发执行
var netProgressMap sync.Map // 存储各子网扫描进度，key为网络ID，value为进度百分比

// ScanView 带进度跟踪的网络扫描请求处理函数，支持扫描状态控制与进度实时更新
func (NetApi) ScanView(c *gin.Context) {
	// 获取请求绑定的网络ID参数
	cr := middleware.GetBind[models.IDRequest](c)
	log := middleware.GetLog(c)

	log.WithFields(map[string]interface{}{
		"net_id": cr.Id,
	}).Info("network scan request received") // 收到网络扫描请求

	// 校验子网是否存在（预加载关联的节点信息）
	var model models.NetModel
	if err := global.DB.Preload("NodeModel").Take(&model, cr.Id).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.Id,
			"error":  err,
		}).Warn("network not found") // 网络不存在
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 校验节点是否运行中（状态1为运行）
	if model.NodeModel.Status != 1 {
		log.WithFields(map[string]interface{}{
			"net_id":    model.ID,
			"node_id":   model.NodeModel.ID,
			"node_name": model.NodeModel.Title,
			"status":    model.NodeModel.Status,
		}).Warn("node is not running") // 节点未运行
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 校验子网可使用IP范围是否为空
	if model.CanUseHoneyIPRange == "" {
		log.WithFields(map[string]interface{}{
			"net_id":  model.ID,
			"network": model.Network,
		}).Warn("network IP range is empty") // 网络可使用ip范围为空
		response.FailWithMsg("网络可使用ip范围为空", c)
		return
	}

	// 校验节点是否在线（获取节点指令通道）
	cmd, ok := grpc_service.GetNodeCommand(model.NodeModel.Uid)
	if !ok {
		log.WithFields(map[string]interface{}{
			"net_id":    model.ID,
			"node_uid":  model.NodeModel.Uid,
			"node_name": model.NodeModel.Title,
		}).Warn("node is offline") // 节点离线
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 获取需过滤的IP列表（已配置的诱捕IP，避免扫描）
	var filterIPList []string
	if err := global.DB.Model(models.HoneyIpModel{}).Where("net_id = ?", cr.Id).Select("ip").Scan(&filterIPList).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id": cr.Id,
			"error":  err,
		}).Error("failed to fetch filtered IP list") // 获取过滤IP列表失败
		response.FailWithMsg("获取过滤IP列表失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"net_id":       model.ID,
		"filtered_ips": len(filterIPList),
	}).Info("fetched filtered IP list for network scan") // 获取过滤IP列表成功

	// 构建扫描指令请求参数
	taskID := fmt.Sprintf("netScan-%d", time.Now().UnixNano()) // 生成唯一任务ID
	req := &node_rpc.CmdRequest{
		CmdType: node_rpc.CmdType_cmdNetScanType,
		TaskID:  taskID,
		NetScanInMessage: &node_rpc.NetScanInMessage{
			Network:      model.Network,            // 子网网段
			IpRange:      model.CanUseHoneyIPRange, // 可扫描IP范围
			FilterIPList: filterIPList,             // 过滤IP列表（诱捕IP）
			NetID:        uint32(model.ID),         // 子网ID（适配gRPC参数类型）
		},
	}

	// 加锁更新子网扫描状态（防止并发扫描）
	mutex.Lock()
	defer mutex.Unlock()

	// 校验是否已有扫描任务在运行（状态2为扫描中）
	if model.ScanStatus == 2 {
		log.WithFields(map[string]interface{}{
			"net_id":      model.ID,
			"scan_status": model.ScanStatus,
			"task_id":     taskID,
		}).Warn("network is already being scanned") // 当前子网正在扫描中
		response.FailWithMsg("当前子网正在扫描中", c)
		return
	}

	// 更新子网扫描状态为“扫描中”
	if err := global.DB.Model(&model).Update("scan_status", 2).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"net_id":     model.ID,
			"new_status": 2,
			"task_id":    taskID,
			"error":      err,
		}).Error("failed to update network scan status") // 更新子网扫描状态失败
		response.FailWithMsg("更新扫描状态失败", c)
		return
	}

	// 下发扫描指令至节点（非阻塞发送，防止通道阻塞）
	select {
	case cmd.ReqChan <- req:
		log.WithFields(map[string]interface{}{
			"net_id":      model.ID,
			"node_uid":    model.NodeModel.Uid,
			"task_id":     taskID,
			"ip_range":    model.CanUseHoneyIPRange,
			"filter_size": len(filterIPList),
		}).Info("scan command sent to node") // 扫描命令发送到节点成功
	default:
		log.WithFields(map[string]interface{}{
			"net_id":   model.ID,
			"node_uid": model.NodeModel.Uid,
			"task_id":  taskID,
			"error":    "command channel is busy",
		}).Warn("failed to send scan command") // 扫描命令发送到节点失败
		response.FailWithMsg("发送命令通道繁忙", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"net_id":  model.ID,
		"task_id": taskID,
	}).Info("scan task started successfully") // 扫描任务启动成功

	// 返回扫描任务启动成功响应
	response.Ok(map[string]string{
		"task_id": taskID,
		"message": "扫描任务已启动，请稍后查询结果",
	}, "扫描任务已启动", c)

	// 异步处理扫描结果（独立协程，避免阻塞HTTP响应）
	go func(nodeUid string, netModel models.NetModel, cmdChan *grpc_service.Command, taskID string) {
		// 设置5分钟超时上下文，防止协程泄漏
		ctxAsync, cancelAsync := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancelAsync()

		log := log.WithFields(map[string]interface{}{
			"net_id":   netModel.ID,
			"node_uid": nodeUid,
			"task_id":  taskID,
		})

		log.Info("async scan result processor started") // 异步扫描结果处理器启动

		var netScanMsg []*node_rpc.NetScanOutMessage // 存储扫描结果消息
	label:
		for {
			select {
			// 接收节点返回的扫描结果
			case res := <-cmdChan.ResChan:
				// 过滤非当前任务的响应（放回通道，供其他任务处理）
				if res.TaskID != taskID {
					log.WithFields(map[string]interface{}{
						"received_task_id": res.TaskID,
					}).Debug("discarding response for different task") // 过滤非当前任务的响应
					select {
					case cmdChan.ResChan <- res:
					case <-ctxAsync.Done():
						break label
					}
					continue
				}

				log.WithFields(map[string]interface{}{
					"progress": res.NetScanOutMessage.Progress,
				}).Debug("received scan progress update") // 接收到扫描进度更新

				message := res.NetScanOutMessage

				// 扫描出错时终止循环
				if message.ErrMsg != "" {
					log.WithFields(map[string]interface{}{
						"error_message": message.ErrMsg,
					}).Error("scan error received from node") // 接收到扫描错误
					break label
				}

				// 扫描完成时终止循环
				if message.End {
					log.Info("received scan completion signal") // 接收到扫描完成信号
					break label
				}

				// 收集有效主机信息，更新扫描进度
				if message.Ip != "" {
					netScanMsg = append(netScanMsg, message)
					netProgressMap.Store(uint(message.NetID), float64(message.Progress))
					log.WithFields(map[string]interface{}{
						"ip":       message.Ip,
						"mac":      message.Mac,
						"manuf":    message.Manuf,
						"progress": message.Progress,
					}).Info("discovered host during scan") // 发现主机
				}

			// 超时终止扫描结果处理
			case <-ctxAsync.Done():
				log.WithFields(map[string]interface{}{
					"error": ctxAsync.Err(),
				}).Error("scan operation timed out") // 扫描操作超时
				return
			}
		}

		// 发送扫描结果更新消息
		mq_service.SendWsMsg(mq_service.WsMsgType{
			Type:   2,
			NetID:  netModel.ID,
			NodeID: netModel.NodeID,
		})

		// 处理有效扫描结果（非超时/非错误场景）
		if len(netScanMsg) > 0 || ctxAsync.Err() == nil {
			log.WithFields(map[string]interface{}{
				"host_count": len(netScanMsg),
			}).Info("processing scan results") // 处理扫描结果
			processScanResult(netModel, netScanMsg, log.Data["logID"].(string))
		} else {
			log.Warn("no scan results received") // 未接收到扫描结果
		}

	}(model.NodeModel.Uid, model, cmd, taskID)
}

// processScanResult 处理子网扫描结果，同步更新数据库中的主机信息
func processScanResult(netModel models.NetModel, scanMsgs []*node_rpc.NetScanOutMessage, logID string) {
	log := core.GetLogger().WithField("logID", logID)

	log.Info("starting scan result processing") // 开始处理扫描结果

	// 延迟更新子网扫描状态为“完成”，清理进度缓存
	defer func() {
		if err := global.DB.Model(&netModel).Updates(map[string]any{
			"scan_progress": 100, // 扫描进度置为100%
			"scan_status":   1,   // 扫描状态置为完成（1为完成）
		}).Error; err != nil {
			log.WithFields(map[string]interface{}{
				"error": err,
			}).Error("failed to update network status after scan") // 更新子网扫描状态失败
		} else {
			log.Info("network status updated to completed") // 子网扫描状态更新为完成
		}
		netProgressMap.Delete(netModel.ID) // 清理扫描进度缓存
	}()

	// 查询子网下现有主机记录
	var hostList []models.HostModel
	if err := global.DB.Find(&hostList, "net_id = ?", netModel.ID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("failed to fetch existing hosts for network") // 获取子网下现有主机记录失败
		return
	}

	log.WithFields(map[string]interface{}{
		"existing_hosts": len(hostList),
	}).Info("fetched existing hosts for comparison") // 获取子网现有主机记录

	// 构建现有主机的IP映射
	dbHostMap := make(map[string]models.HostModel)
	for _, host := range hostList {
		dbHostMap[host.IP] = host
	}

	// 构建扫描结果的IP映射
	scanResultMap := make(map[string]*node_rpc.NetScanOutMessage)
	for _, msg := range scanMsgs {
		if msg.Ip != "" {
			scanResultMap[msg.Ip] = msg
		}
	}

	// 分类处理扫描结果：新增/更新/删除主机
	var newHosts []models.HostModel     // 新增主机列表
	var deletedHostIDs []uint           // 待删除主机ID列表
	var updatedHosts []models.HostModel // 待更新主机列表

	// 处理扫描到的主机（新增/更新）
	for ip, scanMsg := range scanResultMap {
		if dbHost, exists := dbHostMap[ip]; exists {
			// 主机已存在，校验MAC/厂商信息是否变更
			if dbHost.Mac != scanMsg.Mac || dbHost.Manuf != scanMsg.Manuf {
				dbHost.Mac = scanMsg.Mac
				dbHost.Manuf = scanMsg.Manuf
				updatedHosts = append(updatedHosts, dbHost)
				log.WithFields(map[string]interface{}{
					"host_ip":   ip,
					"old_mac":   dbHost.Mac,
					"new_mac":   scanMsg.Mac,
					"old_manuf": dbHost.Manuf,
					"new_manuf": scanMsg.Manuf,
				}).Info("host information updated") // 更新主机信息
			}
			delete(dbHostMap, ip) // 移除已处理的主机，剩余为失联主机
		} else {
			// 主机不存在，加入新增列表
			newHosts = append(newHosts, models.HostModel{
				NodeID: netModel.NodeModel.ID,
				NetID:  netModel.ID,
				IP:     scanMsg.Ip,
				Mac:    scanMsg.Mac,
				Manuf:  scanMsg.Manuf,
			})
			log.WithFields(map[string]interface{}{
				"host_ip":  scanMsg.Ip,
				"host_mac": scanMsg.Mac,
				"manuf":    scanMsg.Manuf,
			}).Info("new host discovered") // 发现新主机
		}
	}

	// 处理失联主机（扫描结果中不存在的现有主机）
	for _, dbHost := range dbHostMap {
		deletedHostIDs = append(deletedHostIDs, dbHost.ID)
		log.WithFields(map[string]interface{}{
			"host_id": dbHost.ID,
			"host_ip": dbHost.IP,
		}).Info("host marked for deletion") // 标记主机为删除
	}

	log.WithFields(map[string]interface{}{
		"new_hosts":     len(newHosts),
		"updated_hosts": len(updatedHosts),
		"deleted_hosts": len(deletedHostIDs),
	}).Info("scan result analysis completed") // 扫描结果分析完成

	// 事务执行数据库操作（保证原子性）
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 新增主机
		if len(newHosts) > 0 {
			if err := tx.Create(&newHosts).Error; err != nil {
				log.WithFields(map[string]interface{}{
					"error": err,
				}).Error("failed to create new hosts") // 新增主机失败
				return fmt.Errorf("创建新主机失败: %w", err)
			}
			log.WithFields(map[string]interface{}{
				"count": len(newHosts),
			}).Info("new hosts created in database") // 新主机创建数据库
		}

		// 更新主机信息
		if len(updatedHosts) > 0 {
			for _, host := range updatedHosts {
				if err := tx.Model(&models.HostModel{}).
					Where("id = ?", host.ID).
					Updates(map[string]interface{}{
						"mac":   host.Mac,
						"manuf": host.Manuf,
					}).Error; err != nil {
					log.WithFields(map[string]interface{}{
						"host_id": host.ID,
						"error":   err,
					}).Error("failed to update host information") // 更新主机信息失败
					return fmt.Errorf("更新主机信息失败: %w", err)
				}
			}
			log.WithFields(map[string]interface{}{
				"count": len(updatedHosts),
			}).Info("hosts updated in database") // 主机信息更新数据库
		}

		// 删除失联主机
		if len(deletedHostIDs) > 0 {
			if err := tx.Delete(&models.HostModel{}, deletedHostIDs).Error; err != nil {
				log.WithFields(map[string]interface{}{
					"count": len(deletedHostIDs),
					"error": err,
				}).Error("failed to delete hosts") // 删除主机失败
				return fmt.Errorf("删除主机失败: %w", err)
			}
			log.WithFields(map[string]interface{}{
				"count": len(deletedHostIDs),
			}).Info("hosts deleted from database") // 主机从数据库删除
		}

		// 计算增加和删除的个数，如果有变化就同步到子网表中
		if len(newHosts)-len(deletedHostIDs) != 0 {
			tx.Model(&netModel).Update("host_count", gorm.Expr("host_count + ?", len(newHosts)-len(deletedHostIDs)))
		}

		return nil
	})

	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err,
		}).Error("database transaction failed during result processing") // 数据库事务处理结果失败
	} else {
		log.Info("database updated successfully with scan results") // 数据库成功更新扫描结果
	}
}
