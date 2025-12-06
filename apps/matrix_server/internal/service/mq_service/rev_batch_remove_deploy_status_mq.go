package mq_service

// File: matrix_server/service/mq_service/rev_batch_remove_deploy_status_mq.go
// Description: 批量删除部署状态消息消费模块，实现MQ消息消费、删除部署状态解析、进度计算、IP关联数据清理及分布式锁释放等核心逻辑

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"matrix_server/internal/global"
	"matrix_server/internal/models"

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

		// 计算删除部署进度：移除当前IP的部署记录，统计剩余待处理IP数量
		key := fmt.Sprintf("deploy_create_%d", data.NetID)
		global.Redis.HDel(context.Background(), key, data.IP)         // 删除当前IP的部署状态记录
		maps := global.Redis.HGetAll(context.Background(), key).Val() // 获取子网下剩余待处理IP列表
		remainingQuantity := len(maps)                                // 剩余待处理IP数量
		logrus.Infof("子网 %d 剩余个数 %d", data.NetID, remainingQuantity)

		// 清理当前IP的关联数据：删除关联端口记录及IP本身
		var model models.HoneyIpModel
		err = global.DB.Preload("PortList").Take(&model, "net_id = ? and ip = ?", data.NetID, data.IP).Error
		if err == nil {
			// 删除IP关联的端口记录
			if len(model.PortList) > 0 {
				global.DB.Delete(&model.PortList)
				logrus.Infof("删除关联端口 %d", len(model.PortList))
			}
			// 删除IP记录
			global.DB.Delete(&model)
			logrus.Infof("删除ip %s", model.IP)
		}

		// 判定子网下所有IP删除部署完成：剩余待处理IP数量为0时释放分布式锁并清理Redis键
		if remainingQuantity == 0 {
			// 构建子网部署操作的分布式锁Key
			mutexname := fmt.Sprintf("deploy_action_lock_%d", data.NetID)
			global.Redis.Del(context.Background(), mutexname) // 释放分布式锁
			global.Redis.Del(context.Background(), key)       // 清理子网部署状态Redis键
			logrus.Infof("子网更新部署完成 解锁")
		}
	}
}
