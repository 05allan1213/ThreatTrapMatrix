package mq_service

// File: honey_node/service/mq_service/rev_batch_update_deploy_status_mq.go
// Description: 批量更新部署状态消息消费模块，实现MQ消息消费、更新部署状态解析、子网更新进度统计、端口状态更新及分布式锁释放等核心逻辑

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"matrix_server/internal/global"
	"matrix_server/internal/models"
	"matrix_server/internal/service/redis_service/net_progress"

	"github.com/sirupsen/logrus"
)

// UpdateDeployStatusRequest 批量更新部署状态的MQ消息结构体
type UpdateDeployStatusRequest struct {
	NetID    uint         `json:"netID"`    // 子网ID，关联目标操作子网
	IP       string       `json:"ip"`       // 执行更新部署操作的IP地址
	LogID    string       `json:"logID"`    // 日志ID，用于关联操作全链路日志
	ErrorMsg string       `json:"errorMsg"` // IP级更新部署执行失败时的错误信息
	PortList []PortStatus `json:"portList"` // 该IP下端口更新部署的状态列表
}

// PortStatus 端口更新部署状态结构体
type PortStatus struct {
	Port     int    `json:"port"`     // 执行更新部署操作的端口号
	ErrorMsg string `json:"errorMsg"` // 端口级更新部署执行失败时的错误信息
}

// RevBatchUpdateDeployStatusMq 消费批量更新部署状态的MQ消息
func RevBatchUpdateDeployStatusMq() {
	// 获取全局MQ配置信息
	cfg := global.Config.MQ
	// 注册MQ消费者，监听批量更新部署状态反馈队列
	msgs, err := global.Queue.Consume(
		cfg.BatchUpdateDeployStatusTopic, // 消费的队列名称
		"",                               // 消费者标识（空表示由MQ自动分配）
		true,                             // 自动确认消息（消费后自动告知MQ已处理）
		false,                            // 排他性（false表示非排他消费）
		false,                            // 非本地（false表示接收本地发布的消息）
		false,                            // 非阻塞（false表示阻塞等待消息）
		nil,                              // 额外配置参数
	)
	if err != nil {
		log.Fatalf("无法注册消费者: %v", err)
	}

	// 循环监听并处理MQ消息
	for d := range msgs {
		// 解析消息体到更新部署状态结构体
		var data UpdateDeployStatusRequest
		err = json.Unmarshal(d.Body, &data)
		if err != nil {
			logrus.Errorf("消息格式解析失败 %s %s", err, d.Body)
			continue
		}
		logrus.Infof("接收批量更新部署回调 %s", d.Body)

		// 获取当前子网的更新部署进度信息
		netDeployInfo, err := net_progress.Get(data.NetID)
		if err != nil {
			logrus.Errorf("获取子网部署信息失败 %s", err)
			continue
		}

		// 更新更新部署进度：已完成数+1，若IP级执行出错则更新错误统计
		netDeployInfo.CompletedCount++
		if data.ErrorMsg != "" {
			netDeployInfo.ErrorCount++
			netDeployInfo.ErrorIpList = append(netDeployInfo.ErrorIpList, net_progress.ErrorIp{
				Ip:  data.IP,
				Msg: data.ErrorMsg,
			})
		}

		// 计算并打印当前子网更新部署进度（百分比）
		logrus.Infof("%d当前子网，正在更新部署%d个，共%d个 进度：%.2f%%", data.NetID,
			netDeployInfo.CompletedCount,
			netDeployInfo.AllCount,
			(float64(netDeployInfo.CompletedCount)/float64(netDeployInfo.AllCount))*100,
		)

		// 保存更新后的子网更新部署进度信息到Redis
		err = net_progress.Set(data.NetID, netDeployInfo)
		if err != nil {
			logrus.Errorf("设置子网更新部署信息失败 %s", err)
			continue
		}

		// 遍历端口状态列表，处理端口级更新失败的情况
		for _, status := range data.PortList {
			if status.ErrorMsg != "" {
				// 记录端口绑定失败日志
				logrus.Errorf("端口绑定失败 %s %d %s", data.IP, status.Port, status.ErrorMsg)
				// 更新端口状态为绑定失败（状态2）
				var portModel models.HoneyPortModel
				global.DB.Take(&portModel, "net_id = ? and ip = ? and port = ?", data.NetID, data.IP, status.Port).Update("status", 2)
			}
		}

		// 判定子网更新部署完成：已完成数等于总数时释放分布式锁
		if netDeployInfo.CompletedCount == netDeployInfo.AllCount {
			// 构建子网部署操作的分布式锁Key
			mutexname := fmt.Sprintf("deploy_action_lock_%d", data.NetID)
			global.Redis.Del(context.Background(), mutexname) // 释放分布式锁
			logrus.Infof("子网更新部署完成 解锁")
		}
	}
}
