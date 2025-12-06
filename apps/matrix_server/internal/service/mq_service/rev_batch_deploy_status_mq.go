package mq_service

// File: matrix_server/service/mq_service/rev_batch_deploy_status_mq.go
// Description: 批量部署状态消息消费模块，实现MQ消息消费、部署状态解析、子网部署进度统计、IP记录更新、存活主机入库及分布式锁释放等核心逻辑

import (
	"encoding/json"
	"fmt"
	"log"
	"matrix_server/internal/global"
	"matrix_server/internal/models"
	"matrix_server/internal/service/redis_service/net_lock"
	"matrix_server/internal/service/redis_service/net_progress"

	"github.com/sirupsen/logrus"
)

// DeployStatusRequest 批量部署状态的MQ消息结构体
type DeployStatusRequest struct {
	NetID    uint   `json:"netID"`    // 子网ID，关联目标操作子网
	IP       string `json:"ip"`       // 执行部署操作的IP地址
	Mac      string `json:"mac"`      // IP对应的MAC地址
	LinkName string `json:"linkName"` // 网络接口名称
	LogID    string `json:"logID"`    // 日志ID，用于关联操作全链路日志
	ErrorMsg string `json:"errorMsg"` // 部署执行失败时的错误信息
	Manuf    string `json:"manuf"`    // 设备厂商名称
}

// RevBatchDeployStatusMq 消费批量部署状态的MQ消息
func RevBatchDeployStatusMq() {
	// 获取全局MQ配置信息
	cfg := global.Config.MQ
	// 注册MQ消费者，监听批量部署状态反馈队列
	msgs, err := global.Queue.Consume(
		cfg.BatchDeployStatusTopic, // 消费的队列名称
		"",                         // 消费者标识（空表示由MQ自动分配）
		true,                       // 自动确认消息（消费后自动告知MQ已处理）
		false,                      // 排他性（false表示非排他消费）
		false,                      // 非本地（false表示接收本地发布的消息）
		false,                      // 非阻塞（false表示阻塞等待消息）
		nil,                        // 额外配置参数
	)
	if err != nil {
		log.Fatalf("无法注册消费者: %v", err)
	}

	// 循环监听并处理MQ消息
	for d := range msgs {
		// 解析消息体到部署状态结构体
		var data DeployStatusRequest
		err = json.Unmarshal(d.Body, &data)
		if err != nil {
			logrus.Errorf("消息格式解析失败 %s %s", err, d.Body)
			continue
		}
		logrus.Infof("接收批量部署回调 %s", d.Body)

		// 获取当前子网的部署进度信息
		netDeployInfo, err := net_progress.Get(data.NetID)
		if err != nil {
			logrus.Errorf("获取子网部署信息失败 %s", err)
			continue
		}

		// 更新部署进度：已完成数+1，若有错误则更新错误统计
		netDeployInfo.CompletedCount++
		if data.ErrorMsg != "" {
			netDeployInfo.ErrorCount++
			netDeployInfo.ErrorIpList = append(netDeployInfo.ErrorIpList, net_progress.ErrorIp{
				Ip:  data.IP,
				Msg: data.ErrorMsg,
			})
		}

		// 计算并打印当前子网部署进度（百分比）
		logrus.Infof("%d当前子网，正在部署%d个，共%d个 进度：%.2f%%", data.NetID,
			netDeployInfo.CompletedCount,
			netDeployInfo.AllCount,
			(float64(netDeployInfo.CompletedCount)/float64(netDeployInfo.AllCount))*100,
		)

		// 保存更新后的子网部署进度信息到Redis
		err = net_progress.Set(data.NetID, netDeployInfo)
		if err != nil {
			logrus.Errorf("设置子网部署信息失败 %s", err)
			continue
		}

		// 查询当前IP对应的蜜罐IP记录
		var honeyIp models.HoneyIpModel
		err = global.DB.Take(&honeyIp, "net_id = ? and ip = ?", data.NetID, data.IP).Error
		if err != nil {
			logrus.Errorf("honeyIp记录不存在 %s", d.Body)
			continue
		}

		// 处理存活主机场景：将存活主机信息入库，并删除对应的蜜罐IP记录
		if data.ErrorMsg == "存活主机" {
			global.DB.Create(&models.HostModel{
				NodeID: honeyIp.NodeID,
				NetID:  data.NetID,
				IP:     data.IP,
				Mac:    data.Mac,
				Manuf:  data.Manuf,
			})
			global.DB.Delete(&honeyIp)
			continue
		}

		// 组装蜜罐IP记录更新数据：更新MAC、接口名、错误信息及部署状态
		var hp = models.HoneyIpModel{
			Mac:      data.Mac,
			Network:  data.LinkName,
			ErrorMsg: data.ErrorMsg,
			Status:   2, // 状态2：部署成功
		}
		// 部署失败时更新状态为3（部署失败）
		if data.ErrorMsg != "" {
			hp.Status = 3
		}

		// 更新蜜罐IP记录
		err = global.DB.Model(&honeyIp).Updates(hp).Error
		if err != nil {
			logrus.Errorf("记录更新失败 %s %s", err, d.Body)
		}

		// 判定子网部署完成：已完成数等于总数时释放分布式锁
		if netDeployInfo.CompletedCount == netDeployInfo.AllCount {
			ok, err := net_lock.UnLock(data.NetID)
			fmt.Println(ok, err)
			logrus.Infof("子网部署完成 解锁")
		}
	}

	// 消费者异常退出时记录错误日志
	logrus.Errorf("接收批量部署回调消费者结束")
}
