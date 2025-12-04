package mq_service

// File: matrix_server/service/mq_service/rev_alert_mq.go
// Description: 实现批量部署状态MQ消息的消费逻辑，处理节点上报的部署状态数据，更新诱捕IP记录状态，并在部署完成时释放子网分布式锁

import (
	"encoding/json"
	"fmt"
	"log"
	"matrix_server/internal/global"
	"matrix_server/internal/models"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/sirupsen/logrus"
)

// DeployStatusRequest 部署状态上报消息结构体
type DeployStatusRequest struct {
	NetID    uint    `json:"netID"`    // 子网ID，标识部署所属的子网
	IP       string  `json:"ip"`       // 部署的诱捕IP地址
	Mac      string  `json:"mac"`      // IP绑定的MAC地址
	LinkName string  `json:"linkName"` // 网络接口名称
	LogID    string  `json:"logID"`    // 日志ID，用于关联部署操作的日志记录
	ErrorMsg string  `json:"errorMsg"` // 部署失败时的错误信息，成功时为空
	Progress float64 `json:"progress"` // 部署进度（1-100的小数，100表示部署完成）
}

// RevBatchDeployStatusMq 消费批量部署状态MQ消息
func RevBatchDeployStatusMq() {
	// 获取全局配置中的MQ配置信息
	cfg := global.Config.MQ
	// 注册MQ消费者，监听批量部署状态主题队列
	msgs, err := global.Queue.Consume(
		cfg.BatchDeployStatusTopic, // 消费的队列名称
		"",                         // 消费者标识
		true,                       // 自动确认消息
		false,                      // 非排他性消费
		false,                      // 非本地消费
		false,                      // 非阻塞模式
		nil,                        // 额外消费参数
	)
	if err != nil {
		log.Fatalf("无法注册消费者: %v", err)
	}

	// 循环监听并处理MQ消息
	for d := range msgs {
		// 解析消息体为部署状态结构体
		var data DeployStatusRequest
		err = json.Unmarshal(d.Body, &data)
		if err != nil {
			logrus.Errorf("消息格式解析失败 %s %s", err, d.Body)
			continue // 解析失败跳过当前消息
		}

		// 查询该子网下对应IP的诱捕IP记录
		var honeyIp models.HoneyIpModel
		err = global.DB.Take(&honeyIp, "net_id = ? and ip = ?", data.NetID, data.IP).Error
		if err != nil {
			logrus.Errorf("honeyIp记录不存在 %s", d.Body)
			continue // 记录不存在跳过当前消息
		}

		// 构建诱捕IP记录更新数据
		var hp = models.HoneyIpModel{
			Mac:      data.Mac,      // 更新IP绑定的MAC地址
			Network:  data.LinkName, // 更新网络接口名称
			ErrorMsg: data.ErrorMsg, // 更新部署错误信息
			Status:   2,             // 默认状态：部署成功（2）
		}
		// 若存在错误信息，标记状态为部署失败（3）
		if data.ErrorMsg != "" {
			hp.Status = 3
		}

		// 更新诱捕IP记录的状态及相关信息
		err = global.DB.Model(&honeyIp).Updates(hp).Error
		if err != nil {
			logrus.Errorf("记录更新失败 %s %s", err, d.Body)
		}

		// 部署进度为100表示当前子网部署完成，释放分布式锁
		if data.Progress == 100 {
			logrus.Infof("子网%d部署完成，开始释放分布式锁", data.NetID)
			// 创建redsync的Redis连接池
			pool := goredis.NewPool(global.Redis)
			// 初始化redsync实例
			rs := redsync.New(pool)
			// 构建子网部署锁的key（与部署时的锁key保持一致）
			mutexname := fmt.Sprintf("deploy_create_lock_%d", data.NetID)
			// 创建基于该key的互斥锁（配置与部署时一致）
			mutex := rs.NewMutex(mutexname,
				redsync.WithExpiry(20*time.Minute),           // 锁过期时间20分钟
				redsync.WithTries(1),                         // 重试次数1次
				redsync.WithRetryDelay(500*time.Millisecond), // 重试间隔500毫秒
			)
			// 释放子网部署的分布式锁
			mutex.Unlock()
			logrus.Infof("子网%d部署完成，分布式锁已解锁", data.NetID)
		}
	}
}
