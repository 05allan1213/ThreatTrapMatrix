package mq_service

// File: matrix_server/service/mq_service/send_batch_update_deploy_msg.go
// Description: 实现批量更新部署指令的MQ发送功能，用于将子网诱捕IP的部署配置更新指令下发至指定节点的MQ队列

import (
	"encoding/json"
	"matrix_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// BatchUpdateDeployRequest 批量更新部署指令发送请求结构体
type BatchUpdateDeployRequest struct {
	NetID    uint       `json:"netID"`    // 子网ID，标识更新操作所属的子网
	LogID    string     `json:"logID"`    // 日志ID，用于关联更新操作的日志记录
	IpList   []string   `json:"ipList"`   // 待更新配置的诱捕IP列表（主机模板变更的IP）
	PortList []PortInfo `json:"portList"` // 新的端口转发配置列表（基于新主机模板）
}

// SendBatchUpdateDeployMsg 发送批量更新部署指令到MQ队列
func SendBatchUpdateDeployMsg(nodeUID string, req BatchUpdateDeployRequest) (err error) {
	// 将批量更新部署指令序列化为JSON字节数组
	byteData, _ := json.Marshal(req)
	// 获取全局配置中的MQ配置信息
	cfg := global.Config.MQ

	// 发布批量更新部署指令到MQ交换机
	err = global.Queue.Publish(
		cfg.BatchUpdateDeployExchangeName, // exchange：批量更新部署专用交换机名称
		nodeUID,                           // routing key：目标节点UID，用于精准路由到对应节点
		false,                             // mandatory：不强制要求消息必须路由到队列
		false,                             // immediate：不要求立即投递
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 消息体：序列化后的批量更新部署指令
		})

	// 处理消息发送失败的情况，记录错误日志并返回错误
	if err != nil {
		logrus.Errorf("消息发送失败 %s %s", err, string(byteData))
		return err
	}
	logrus.Infof("消息发送成功 %s", string(byteData))
	return
}
