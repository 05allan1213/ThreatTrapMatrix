package mq_service

// File: matrix_server/service/mq_service/rev_batch_deploy_status_mq.go
// Description: 实现批量部署状态MQ消息的消费逻辑，处理节点上报的部署状态数据，更新诱捕IP记录状态，并在部署完成时释放子网分布式锁

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"matrix_server/internal/global"
	"matrix_server/internal/models"

	"github.com/sirupsen/logrus"
)

// DeployStatusRequest 部署状态上报消息结构体
type DeployStatusRequest struct {
	NetID    uint   `json:"netID"`    // 子网ID，标识部署所属的子网
	IP       string `json:"ip"`       // 部署的诱捕IP地址
	Mac      string `json:"mac"`      // IP绑定的MAC地址
	LinkName string `json:"linkName"` // 网络接口名称
	LogID    string `json:"logID"`    // 日志ID，用于关联部署操作的日志记录
	ErrorMsg string `json:"errorMsg"` // 部署失败时的错误信息，成功时为空
	Manuf    string `json:"manuf"`    // 厂商信息
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
		logrus.Infof("收到批量部署回调 %s", d.Body)

		// 计算部署进度
		key := fmt.Sprintf("deploy_create_%d", data.NetID)
		global.Redis.HDel(context.Background(), key, data.IP)
		maps := global.Redis.HGetAll(context.Background(), key).Val()
		remainingQuantity := len(maps) // 剩余个数
		logrus.Infof("子网 %d 剩余个数 %d", data.NetID, remainingQuantity)

		// 查询该子网下对应IP的诱捕IP记录
		var honeyIp models.HoneyIpModel
		err = global.DB.Take(&honeyIp, "net_id = ? and ip = ?", data.NetID, data.IP).Error
		if err != nil {
			logrus.Errorf("honeyIp记录不存在 %s", d.Body)
			continue // 记录不存在跳过当前消息
		}

		if data.ErrorMsg == "存活主机" {
			// 入存活主机的库
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

		// 如何计算，当前子网部署完了？
		if remainingQuantity == 0 {
			// 构建子网部署锁的key（与部署时的锁key保持一致）
			mutexname := fmt.Sprintf("deploy_action_lock_%d", data.NetID)
			global.Redis.Del(context.Background(), mutexname)
			key := fmt.Sprintf("deploy_create_%d", data.NetID)
			global.Redis.Del(context.Background(), key)
			logrus.Infof("子网%d部署完成，分布式锁已解锁", data.NetID)
		}
	}
}
