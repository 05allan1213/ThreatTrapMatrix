package mq_service

// File: honey_node/service/mq_service/batch_deploy_exchange.go
// Description: 实现诱捕IP批量部署的MQ消费逻辑，包含IP配置创建、端口转发配置、数据持久化及异步任务控制等核心功能

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

	"github.com/sirupsen/logrus"
)

// BatchDeployRequest MQ消费的批量部署请求结构体
type BatchDeployRequest struct {
	NetID   uint       `json:"netID"`   // 子网ID
	LogID   string     `json:"logID"`   // 日志ID
	Network string     `json:"network"` // 网卡名称
	IPList  []DeployIp `json:"ipList"`  // 待部署IP列表
}

// DeployIp 单IP部署配置信息结构体
type DeployIp struct {
	Ip       string     `json:"ip"`       // 待部署的诱捕IP地址
	IsTan    bool       `json:"isTan"`    // 是否为探针IP标识
	Mask     int8       `json:"mask"`     // IP子网掩码
	PortList []PortInfo `json:"portList"` // 该IP关联的端口转发配置列表
}

// BatchDeployExChange 批量部署MQ消息处理函数
func BatchDeployExChange(msg string) error {
	// 解析MQ消息为批量部署请求结构体
	var req BatchDeployRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		logrus.Errorf("JSON解析失败: %v, 消息: %s", err, msg)
		return nil // 保持原有逻辑，解析失败时返回nil
	}

	// 协程池控制：限制并发数为100
	var maxChan = make(chan struct{}, 100)
	// 等待组：用于等待所有IP创建协程执行完成
	var wait sync.WaitGroup
	// 汇总所有待创建的端口转发配置
	var portList []PortInfo
	// 原子计数器：记录已完成部署的IP数量（保证并发安全）
	var completedCount int64
	// 总部署IP数量，用于计算部署进度
	var allCount = int64(len(req.IPList))
	// 互斥锁：用于保护进度上报数据
	var mutex = sync.Mutex{}

	// 第一步：批量创建诱捕IP配置并持久化数据
	for _, ip := range req.IPList {
		maxChan <- struct{}{} // 协程池占位，控制并发数
		wait.Add(1)           // 增加等待组计数

		// 异步创建单个IP配置
		go func(info DeployIp, wait *sync.WaitGroup) {
			// 协程退出时释放协程池资源、减少等待组计数、更新已完成计数
			defer func() {
				<-maxChan
				wait.Done()
			}()

			// 生成随机网络接口名称（前缀hy_+6位不重复随机字符串）
			linkName := fmt.Sprintf("hy_%s", random.RandStrV2(6))
			// 调用IP服务配置IP地址、掩码及网络接口
			mac, err := ip_service.SetIp(ip_service.SetIpRequest{
				Ip:       info.Ip,
				Mask:     info.Mask,
				LinkName: linkName,
				Network:  req.Network,
			})
			mutex.Lock()
			// 获取当前已完成的IP数量
			currentCompletedCount := atomic.AddInt64(&completedCount, 1)
			// 初始化部署状态请求数据
			res := DeployStatusRequest{
				NetID:    req.NetID,
				IP:       info.Ip,
				Mac:      mac,
				LinkName: linkName,
				LogID:    req.LogID,
				Progress: (float64(currentCompletedCount) / float64(allCount)) * 100, // 计算当前部署进度
			}

			// IP配置失败时记录错误信息
			if err != nil {
				res.ErrorMsg = err.Error()
				SendDeployStatusMsg(res)
				return
			}

			// IP配置成功后持久化数据到数据库
			global.DB.Create(&models.IpModel{
				Ip:       info.Ip,
				Mask:     info.Mask,
				LinkName: linkName,
				Network:  req.Network,
				Mac:      mac,
			})

			// 上报成功部署的IP信息
			SendDeployStatusMsg(res)
			mutex.Unlock()
		}(ip, &wait)

		// 异步汇总当前IP的端口转发配置到全局列表
		go func(info DeployIp) {
			for _, p := range info.PortList {
				portList = append(portList, p)
			}
		}(ip)
	}
	wait.Wait() // 等待所有IP创建协程执行完成

	// 第二步：批量创建端口转发配置并建立隧道
	for _, info := range portList {
		// 持久化端口转发配置到数据库
		global.DB.Create(&models.PortModel{
			TargetAddr: info.TargetAddr(), // 目标地址
			LocalAddr:  info.LocalAddr(),  // 本地地址
		})

		// 异步建立端口隧道
		go func(port PortInfo) {
			err := port_service.Tunnel(port.LocalAddr(), port.TargetAddr())
			if err != nil {
				logrus.Errorf("端口绑定失败 %s", err)
			}
			// 异常说明：端口绑定失败大概率是IP未配置完成，或端口未释放
			// 待补充逻辑：仅通知失败的端口绑定信息给管理端
		}(info)
	}

	return nil
}
