package mq_service

// File: matrix_server/service/mq_service/send_batch_deploy_msg.go
// Description: 实现批量部署指令的MQ发送功能，用于将子网诱捕IP批量部署指令下发至指定节点的MQ队列

import (
	"encoding/json"
	"matrix_server/internal/global"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// BatchDeployRequest 批量部署指令发送请求结构体
type BatchDeployRequest struct {
	NetID   uint       `json:"netID"`   // 子网ID，标识部署的目标子网
	LogID   string     `json:"logID"`   // 日志ID，用于关联部署操作的日志记录
	Network string     `json:"network"` // 网卡名称，指定IP所属的网络环境
	IPList  []DeployIp `json:"ipList"`  // 待部署IP列表，包含每个IP的具体配置
}

// DeployIp 单IP部署配置结构体
type DeployIp struct {
	Ip       string     `json:"ip"`       // 待部署的诱捕IP地址
	IsTan    bool       `json:"isTan"`    // 是否为探针IP标识
	Mask     int8       `json:"mask"`     // IP子网掩码
	PortList []PortInfo `json:"portList"` // 该IP关联的端口转发配置列表
}

// PortInfo 端口转发配置结构体
type PortInfo struct {
	IP       string `json:"ip"`       // 源IP地址
	Port     int    `json:"port"`     // 源端口号
	DestIP   string `json:"destIP"`   // 目标IP地址
	DestPort int    `json:"destPort"` // 目标端口号
}

// SendBatchDeployMsg 发送批量部署指令到MQ队列
func SendBatchDeployMsg(nodeUID string, req BatchDeployRequest) (err error) {
	// 将批量部署指令序列化为JSON字节数组
	byteData, _ := json.Marshal(req)
	// 获取全局配置中的MQ配置信息
	cfg := global.Config.MQ

	// 发布批量部署指令到MQ交换机
	err = global.Queue.Publish(
		cfg.BatchDeployExchangeName, // exchange：批量部署专用交换机名称
		nodeUID,                     // routing key：目标节点UID，用于精准路由到对应节点
		false,                       // mandatory：不强制要求消息必须路由到队列
		false,                       // immediate：不要求立即投递
		amqp.Publishing{
			ContentType: "text/plain", // 消息内容类型
			Body:        byteData,     // 消息体：序列化后的批量部署指令
		})

	// 处理消息发送失败的情况
	if err != nil {
		logrus.Errorf("消息发送失败 %s %s", err, string(byteData))
		return err
	}

	return
}
