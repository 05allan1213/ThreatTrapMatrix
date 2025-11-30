package mq_service

// File: honey_server/mq_service/register_exchange.go
// Description: RabbitMQ交换器声明服务，负责初始化系统所需的各类交换器（Exchange）

import (
	"honey_server/internal/global"

	"github.com/sirupsen/logrus"
)

// RegisterExChange 注册系统所需的所有RabbitMQ交换器
func RegisterExChange() {
	cfg := global.Config.MQ
	// 声明创建诱捕IP的交换器
	exchangeDeclare(cfg.CreateIpExchangeName)
	// 声明删除诱捕IP的交换器
	exchangeDeclare(cfg.DeleteIpExchangeName)
	// 声明绑定端口的交换器
	exchangeDeclare(cfg.BindPortExchangeName)
}

// exchangeDeclare 声明单个RabbitMQ交换器
func exchangeDeclare(name string) {
	var err error
	// 调用AMQP接口声明交换器
	err = global.Queue.ExchangeDeclare(
		name,     // 交换器名称
		"direct", // 交换器类型：direct（直接交换器），根据路由键精确匹配队列
		true,     // 持久化：交换器在服务器重启后仍然存在
		false,    // 自动删除：当所有绑定队列都不再使用时，交换器不会自动删除
		false,    // 内部：是否为内部交换器（仅用于交换机间转发，不接收客户端消息）
		false,    // 非阻塞：是否阻塞等待声明完成
		nil,      // 额外参数：无特殊配置
	)
	if err != nil {
		logrus.Fatalf("声明交换器 %s 失败 %s", name, err)
		return
	}
	logrus.Infof("声明交换器 %s 成功", name)
}
