package mq_service

// File: honey_node/service/mq_service/rev_batch_update_deploy_status_mq.go
// Description: 实现服务端批量更新部署状态MQ消息的消费逻辑，解析节点上报的IP/端口更新结果，更新端口状态、计算部署进度，在子网更新完成时释放分布式锁

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"matrix_server/internal/global"
	"matrix_server/internal/models"

	"github.com/sirupsen/logrus"
)

// UpdateDeployStatusRequest 服务端接收的更新部署状态上报消息结构体
type UpdateDeployStatusRequest struct {
	NetID    uint         `json:"netID"`    // 子网ID，标识更新操作所属的子网
	IP       string       `json:"ip"`       // 执行更新的诱捕IP地址
	LogID    string       `json:"logID"`    // 日志ID，用于关联更新操作的日志记录
	ErrorMsg string       `json:"errorMsg"` // IP级别的更新错误信息
	PortList []PortStatus `json:"portList"` // 该IP下各端口的更新执行状态列表
}

// PortStatus 端口更新状态信息结构体
type PortStatus struct {
	Port     int    `json:"port"`     // 端口号
	ErrorMsg string `json:"errorMsg"` // 该端口更新失败时的错误信息，成功时为空
}

// RevBatchUpdateDeployStatusMq 消费批量更新部署状态MQ消息
func RevBatchUpdateDeployStatusMq() {
	// 获取全局配置中的MQ配置信息
	cfg := global.Config.MQ
	// 注册MQ消费者，监听批量更新部署状态主题队列
	msgs, err := global.Queue.Consume(
		cfg.BatchUpdateDeployStatusTopic, // 消费的队列名称
		"",                               // 消费者标识
		true,                             // 自动确认消息
		false,                            // 非排他性消费
		false,                            // 非本地消费
		false,                            // 非阻塞模式
		nil,                              // 额外消费参数
	)
	if err != nil {
		log.Fatalf("无法注册消费者: %v", err)
	}

	// 循环监听并处理MQ消息
	for d := range msgs {
		// 解析消息体为更新部署状态结构体
		var data UpdateDeployStatusRequest
		err = json.Unmarshal(d.Body, &data)
		if err != nil {
			logrus.Errorf("消息格式解析失败 %s %s", err, d.Body)
			continue // 解析失败跳过当前消息
		}
		logrus.Infof("接收批量更新部署回调 %s", d.Body)

		// ---------------------- 计算子网更新剩余进度 ----------------------
		// 构建Redis进度统计Key（格式：deploy_create_子网ID），用于记录子网待更新IP列表
		key := fmt.Sprintf("deploy_create_%d", data.NetID)
		// 从Redis中移除当前已完成更新的IP（标记该IP更新完成）
		global.Redis.HDel(context.Background(), key, data.IP)
		// 获取子网剩余待更新的IP列表（计算剩余数量）
		maps := global.Redis.HGetAll(context.Background(), key).Val()
		remainingQuantity := len(maps) // 子网剩余待更新的IP个数
		logrus.Infof("子网 %d 剩余个数 %d", data.NetID, remainingQuantity)

		// ---------------------- 更新端口错误状态 ----------------------
		for _, status := range data.PortList {
			// 端口更新失败时，记录错误并更新端口状态为2（失败）
			if status.ErrorMsg != "" {
				logrus.Errorf("端口绑定失败 %s %d %s", data.IP, status.Port, status.ErrorMsg)
				var portModel models.HoneyPortModel
				// 查询对应端口记录并更新状态为失败（2）
				global.DB.Take(&portModel, "net_id = ? and ip = ? and port = ?", data.NetID, data.IP, status.Port).Update("status", 2)
			}
		}

		// ---------------------- 子网更新完成时释放分布式锁 ----------------------
		if remainingQuantity == 0 {
			// 构建子网更新操作锁的Key（与更新时的锁Key保持一致）
			mutexname := fmt.Sprintf("deploy_action_lock_%d", data.NetID)
			// 删除Redis中的分布式锁（释放锁）
			global.Redis.Del(context.Background(), mutexname)
			// 删除Redis中的进度统计Key（清理缓存）
			global.Redis.Del(context.Background(), key)
			logrus.Infof("子网更新部署完成 解锁")
		}
	}
}
