package mq_service

// File: honey_node/service/mq_service/batch_deploy_exchange.go
// Description: 实现节点侧批量部署MQ消息消费逻辑，包含任务入库、异步创建诱捕IP配置、端口转发配置、部署状态上报及任务状态更新等核心功能

import (
	"encoding/json"
	"fmt"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/ip_service"
	"honey_node/internal/service/port_service"
	"honey_node/internal/utils/random"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BatchDeployExChange 节点侧批量部署MQ消息处理函数
func BatchDeployExChange(msg string) error {
	// 解析MQ消息体为批量部署请求结构体
	var req models.BatchDeployRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		logrus.Errorf("JSON解析失败: %v, 消息: %s", err, msg)
		return nil // 保持原有逻辑，解析失败时返回nil，不中断后续处理
	}

	// 生成唯一任务ID，用于标识本次批量部署任务
	taskID := uuid.New().String()
	// 将批量部署任务入库，记录任务基础信息及原始部署指令
	err := global.DB.Create(&models.TaskModel{
		TaskID:          taskID, // 任务唯一标识
		Type:            1,      // 任务类型：1-批量部署
		BatchDeployData: &req,   // 批量部署原始指令数据
		Status:          0,      // 任务状态：0-执行中
	}).Error
	if err != nil {
		logrus.Errorf("任务入库失败 %s", err)
		return err
	}

	// 异步执行批量部署核心逻辑（非阻塞，避免阻塞MQ消费流程）
	go func(req models.BatchDeployRequest, taskID string) {
		// 协程池控制：控制并发数
		var maxChan = make(chan struct{}, 1)
		// 等待组：用于等待所有IP创建协程执行完成
		var wait sync.WaitGroup
		// 汇总所有待创建的端口转发配置
		var portList []models.PortInfo
		// 原子计数器：记录已完成部署的IP数量（保证并发安全）
		var completedCount int64
		// 总部署IP数量，用于计算部署进度
		var allCount = int64(len(req.IPList))
		// 互斥锁：保证部署状态上报时的数据一致性
		var mutex = sync.Mutex{}

		// 第一步：批量异步创建诱捕IP配置并上报部署状态
		for _, ip := range req.IPList {
			maxChan <- struct{}{} // 协程池占位，控制并发数
			wait.Add(1)           // 增加等待组计数

			// 异步创建单个IP配置
			go func(info models.DeployIp, wait *sync.WaitGroup) {
				// 协程退出时释放协程池资源、减少等待组计数
				defer func() {
					<-maxChan
					wait.Done()
				}()

				// 生成唯一网络接口名称（前缀hy_+6位唯一随机字符串）
				linkName := fmt.Sprintf("hy_%s", random.RandStrV2(6))
				// 调用IP服务配置IP地址、掩码及网络接口
				mac, err := ip_service.SetIp(ip_service.SetIpRequest{
					Ip:       info.Ip,
					Mask:     info.Mask,
					LinkName: linkName,
					Network:  req.Network,
				})
				time.Sleep(1 * time.Second)
				// 加锁保证进度计算和状态上报的原子性
				mutex.Lock()
				// 原子更新已完成部署的IP数量，避免并发计数错误
				currentCompletedCount := atomic.AddInt64(&completedCount, 1)
				// 构建部署状态上报数据
				res := DeployStatusRequest{
					NetID:    req.NetID,
					IP:       info.Ip,
					Mac:      mac,
					LinkName: linkName,
					LogID:    req.LogID,
					Progress: (float64(currentCompletedCount) / float64(allCount)) * 100, // 计算当前部署进度
				}

				// IP配置失败时上报错误状态
				if err != nil {
					res.ErrorMsg = err.Error()
					SendDeployStatusMsg(res)
					mutex.Unlock() // 解锁后退出
					return
				}

				// IP配置成功后持久化IP信息到数据库
				global.DB.Create(&models.IpModel{
					Ip:       info.Ip,
					Mask:     info.Mask,
					LinkName: linkName,
					Network:  req.Network,
					Mac:      mac,
				})

				// 上报IP配置成功的部署状态
				SendDeployStatusMsg(res)
				mutex.Unlock() // 解锁
			}(ip, &wait)

			// 异步汇总当前IP的端口转发配置到全局列表
			go func(info models.DeployIp) {
				for _, p := range info.PortList {
					portList = append(portList, p)
				}
			}(ip)
		}
		wait.Wait() // 等待所有IP创建协程执行完成

		// 第二步：批量创建端口转发配置并异步建立端口隧道
		for _, info := range portList {
			// 持久化端口转发配置到数据库
			global.DB.Create(&models.PortModel{
				TargetAddr: info.TargetAddr(), // 目标地址
				LocalAddr:  info.LocalAddr(),  // 本地地址
			})

			// 异步建立端口隧道
			go func(port models.PortInfo) {
				err := port_service.Tunnel(port.LocalAddr(), port.TargetAddr())
				if err != nil {
					logrus.Errorf("端口绑定失败 %s", err)
				}
				// 异常说明：端口绑定失败大概率是IP未配置完成，或端口未释放
				// 待补充逻辑：仅通知失败的端口绑定信息给服务端
			}(info)
		}

		// 第三步：更新批量部署任务状态为执行完成
		var taskModel models.TaskModel
		err = global.DB.Take(&taskModel, "task_id = ?", taskID).Error
		if err != nil {
			logrus.Errorf("%s 任务不存在", taskID)
			return
		}
		// 更新任务状态为1（执行完成）
		global.DB.Model(&taskModel).Update("status", 1)
	}(req, taskID)

	return nil
}
