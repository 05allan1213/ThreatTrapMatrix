package mq_service

// File: matrix_server/service/mq_service/rev_batch_remove_deploy_status_mq.go
// Description: 批量删除部署状态消息消费模块，实现MQ消息消费、删除部署状态解析、子网删除进度统计、IP关联数据清理及分布式锁释放等核心逻辑

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

// RemoveDeployStatusRequest 批量删除部署状态的MQ消息结构体
type RemoveDeployStatusRequest struct {
	NetID    uint   `json:"netID"`    // 子网ID，关联目标操作子网
	IP       string `json:"ip"`       // 已执行删除部署操作的IP地址
	LogID    string `json:"logID"`    // 日志ID，用于关联操作全链路日志
	ErrorMsg string `json:"errorMsg"` // 删除部署执行失败时的错误信息
}

// RevBatchRemoveDeployStatusMq 消费批量删除部署状态的MQ消息
func RevBatchRemoveDeployStatusMq() {
	// 获取全局MQ配置信息
	cfg := global.Config.MQ
	// 注册MQ消费者，监听批量删除部署状态反馈队列
	msgs, err := global.Queue.Consume(
		cfg.BatchRemoveDeployStatusTopic, // 消费的队列名称
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
		// 解析消息体到删除部署状态结构体
		var data RemoveDeployStatusRequest
		err = json.Unmarshal(d.Body, &data)
		if err != nil {
			logrus.Errorf("消息格式解析失败 %s %s", err, d.Body)
			continue
		}
		logrus.Infof("接收批量删除部署回调 %s", d.Body)

		// 获取当前子网的删除部署进度信息
		netDeployInfo, err := net_progress.Get(data.NetID)
		if err != nil {
			logrus.Errorf("获取子网部署信息失败 %s", err)
			continue
		}

		// 更新删除部署进度：已完成数+1，若有错误则更新错误统计
		netDeployInfo.CompletedCount++
		if data.ErrorMsg != "" {
			netDeployInfo.ErrorCount++
			netDeployInfo.ErrorIpList = append(netDeployInfo.ErrorIpList, net_progress.ErrorIp{
				Ip:  data.IP,
				Msg: data.ErrorMsg,
			})
		}

		// 计算并打印当前子网删除部署进度（百分比）
		logrus.Infof("%d当前子网，正在删除部署%d个，共%d个 进度：%.2f%%", data.NetID,
			netDeployInfo.CompletedCount,
			netDeployInfo.AllCount,
			(float64(netDeployInfo.CompletedCount)/float64(netDeployInfo.AllCount))*100,
		)

		// 保存更新后的子网删除部署进度信息到Redis
		err = net_progress.Set(data.NetID, netDeployInfo)
		if err != nil {
			logrus.Errorf("设置子网删除部署信息失败 %s", err)
			continue
		}

		// 查询当前IP对应的蜜罐IP记录并清理关联数据
		var model models.HoneyIpModel
		err = global.DB.Preload("PortList").Take(&model, "net_id = ? and ip = ?", data.NetID, data.IP).Error
		if err == nil {
			// 删除IP关联的端口记录
			if len(model.PortList) > 0 {
				global.DB.Delete(&model.PortList)
				logrus.Infof("删除关联端口 %d", len(model.PortList))
			}
			// 删除蜜罐IP记录
			global.DB.Delete(&model)
			logrus.Infof("删除ip %d", len(model.IP))
		}

		// 判定子网删除部署完成：已完成数等于总数时释放分布式锁
		if netDeployInfo.CompletedCount == netDeployInfo.AllCount {
			// 构建子网部署操作的分布式锁Key
			mutexname := fmt.Sprintf("deploy_action_lock_%d", data.NetID)
			global.Redis.Del(context.Background(), mutexname) // 释放分布式锁
			logrus.Infof("子网删除部署完成 解锁")
		}
	}
}
