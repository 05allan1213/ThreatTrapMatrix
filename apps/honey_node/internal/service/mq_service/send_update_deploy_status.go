package mq_service

// File: honey_node/service/mq_service/send_update_deploy_status.go
// Description: 实现批量更新部署状态MQ消息的发送功能，上报节点侧诱捕IP端口更新的执行结果（成功/失败）至服务端

import (
	"encoding/json"
	"honey_node/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// UpdateDeployStatusRequest 更新部署状态上报消息结构体
type UpdateDeployStatusRequest struct {
	NetID    uint       `json:"netID"`    // 子网ID，标识更新操作所属的子网
	IP       string     `json:"ip"`       // 执行更新的诱捕IP地址
	LogID    string     `json:"logID"`    // 日志ID，用于关联更新操作的日志记录
	ErrorMsg string     `json:"errorMsg"` // IP级别的更新错误信息（整体失败时填充）
	PortList []PortInfo `json:"portList"` // 该IP下各端口的更新执行状态列表
}

// PortInfo 端口更新状态信息结构体
type PortInfo struct {
	Port     int    `json:"port"`     // 端口号
	ErrorMsg string `json:"errorMsg"` // 该端口更新失败时的错误信息
}

// SendUpdateDeployStatusMsg 发送更新部署状态MQ消息
func SendUpdateDeployStatusMsg(data UpdateDeployStatusRequest) {
	// 将更新状态数据序列化为JSON字节数组
	byteData, _ := json.Marshal(data)
	// 获取全局配置中的MQ配置信息
	cfg := global.Config.MQ

	// 发布MQ消息至批量更新部署状态主题队列
	err := global.Queue.Publish(
		"",                               // exchange：空表示使用默认交换机
		cfg.BatchUpdateDeployStatusTopic, // routing key：批量更新部署状态主题队列名称（精准路由）
		false,                            // mandatory：不强制要求消息必须路由到队列（路由失败则丢弃）
		false,                            // immediate：不要求立即投递（无消费者时不返回错误）
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型：纯文本（JSON格式）
			Body:        byteData,     // 消息体：序列化后的更新状态数据
		})

	// 处理消息发送失败的情况，记录错误日志
	if err != nil {
		logrus.Errorf("发送失败: %v %s", err, string(byteData))
		return
	}
	// 消息发送成功，记录信息日志
	logrus.Infof("发送成功: %s", string(byteData))
}
