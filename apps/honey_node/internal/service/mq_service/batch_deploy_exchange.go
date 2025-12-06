package mq_service

// File: honey_node/service/mq_service/batch_deploy_exchange.go
// Description: 实现节点侧批量部署MQ消息消费逻辑，新增多维度IP合法性校验（探针IP/已部署IP/本地IP/ARP存活），保障部署安全性；包含任务入库、异步IP配置/端口转发、部署状态上报及任务状态更新等核心功能

import (
	"encoding/json"
	"fmt"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/ip_service"
	"honey_node/internal/service/port_service"
	info2 "honey_node/internal/utils/info"
	"honey_node/internal/utils/random"
	"net"
	"sync"

	"github.com/google/uuid"
	"github.com/j-keck/arping"
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
	go DeployTask(req, taskID)
	return nil
}

// DeployTask 批量部署核心任务执行函数
func DeployTask(req models.BatchDeployRequest, taskID string) {
	// 协程池控制：限制IP创建的并发数为200，平衡部署效率与系统资源
	var maxChan = make(chan struct{}, 200)
	// 等待组：用于等待所有IP创建协程执行完成
	var wait sync.WaitGroup
	// 汇总所有待创建的端口转发配置
	var portList []models.PortInfo

	// 第一步：批量异步创建诱捕IP配置（含多维度合法性校验）
	for _, ip := range req.IPList {
		maxChan <- struct{}{} // 协程池占位，控制并发数
		wait.Add(1)           // 增加等待组计数

		// 异步处理单个IP的部署逻辑
		go func(info models.DeployIp, wait *sync.WaitGroup) {
			// 协程退出时释放协程池资源、减少等待组计数
			defer func() {
				<-maxChan
				wait.Done()
			}()

			// 初始化部署状态上报数据（基础字段）
			res := DeployStatusRequest{
				NetID: req.NetID,
				IP:    info.Ip,
				LogID: req.LogID,
			}

			// 校验1：探针IP跳过部署（探针IP为节点核心IP，无需重复创建）
			if info.Ip == req.TanIp {
				res.Mac, _ = ip_service.GetMACAddress(req.Network) // 获取探针IP的MAC地址
				SendDeployStatusMsg(res)                           // 上报探针IP部署状态（无需创建）
				return
			}

			// 校验2：已部署IP跳过部署（数据库中存在则复用原有配置）
			var model models.IpModel
			global.DB.Find(&model, "network = ? and ip = ?", req.Network, info.Ip)
			if model.ID != 0 {
				res.Mac = model.Mac           // 复用已部署IP的MAC地址
				res.LinkName = model.LinkName // 复用已部署IP的网络接口名称
				SendDeployStatusMsg(res)      // 上报已部署IP状态（无需重复创建）
				return
			} else {
				// 校验3：本地IP冲突检测（待部署IP已存在于本地网卡则报错）
				ok := info2.FindLocalIp(info.Ip)
				if ok {
					res.ErrorMsg = fmt.Sprintf("当前ip存在与本地ip中") // 记录本地IP冲突错误
					SendDeployStatusMsg(res)                           // 上报IP冲突状态
					return
				}
			}

			// 校验4：ARP存活检测（待部署IP已被局域网其他主机占用则报错）
			_mac, _, err := arping.PingOverIfaceByName(net.ParseIP(info.Ip), req.Network)
			if err == nil {
				// 查询MAC地址对应的设备厂商信息，便于排查存活主机
				manuf, _ := core.ManufQuery(_mac.String())
				logrus.Warnf("存活主机 %s %s %s", info.Ip, _mac.String(), manuf)

				res.ErrorMsg = "存活主机" // 记录存活主机错误
				res.Mac = _mac.String()   // 记录存活主机的MAC地址
				res.Manuf = manuf         // 记录存活主机的设备厂商
				SendDeployStatusMsg(res)  // 上报存活主机状态
				return
			}

			// 所有校验通过，开始创建诱捕IP配置
			// 生成唯一网络接口名称（前缀hy_+6位唯一随机字符串，避免接口名冲突）
			linkName := fmt.Sprintf("hy_%s", random.RandStrV2(6))
			// 调用IP服务配置IP地址、掩码及网络接口
			mac, err := ip_service.SetIp(ip_service.SetIpRequest{
				Ip:       info.Ip,
				Mask:     info.Mask,
				LinkName: linkName,
				Network:  req.Network,
			})
			// IP配置失败时上报错误状态并退出
			if err != nil {
				res.ErrorMsg = err.Error()
				SendDeployStatusMsg(res)
				return
			}

			// IP配置成功，填充部署状态数据
			res.Mac = mac
			res.LinkName = linkName
			// 持久化IP配置信息到数据库
			global.DB.Create(&models.IpModel{
				Ip:       info.Ip,
				Mask:     info.Mask,
				LinkName: linkName,
				Network:  req.Network,
				Mac:      mac,
			})

			// 上报IP配置成功的部署状态
			SendDeployStatusMsg(res)
		}(ip, &wait)

		// 异步汇总当前IP的端口转发配置到全局列表（不阻塞IP创建流程）
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

		// 异步建立端口隧道（非阻塞，提升部署效率）
		go func(port models.PortInfo) {
			err := port_service.Tunnel(port.LocalAddr(), port.TargetAddr())
			if err != nil {
				logrus.Errorf("端口绑定失败 %s", err)
			}
			// 异常说明：端口绑定失败大概率是IP未配置完成，或端口未释放
			// 待补充逻辑：仅通知失败的端口绑定信息给服务端
		}(info)
	}
	logrus.Infof("批量部署完成")

	// 第三步：更新批量部署任务状态为执行完成
	var taskModel models.TaskModel
	err := global.DB.Take(&taskModel, "task_id = ?", taskID).Error
	if err != nil {
		logrus.Errorf("%s 任务不存在", taskID)
		return
	}
	// 更新任务状态为1（执行完成）
	global.DB.Model(&taskModel).Update("status", 1)
}
