package mq_service

// File: alert_server/service/mq_service/enter.go
// Description: MQ服务启动模块，负责告警队列声明及告警消息消费协程初始化，确保MQ消息接收通道就绪

import (
	"alert_server/internal/global"

	"github.com/sirupsen/logrus"
)

// Run 初始化MQ服务核心流程：声明告警队列 + 启动告警消息消费协程
func Run() {
	cfg := global.Config.Alert
	// 声明MQ告警队列，配置队列基础属性
	_, err := global.Queue.QueueDeclare(
		cfg.AlertTopic, // 队列名称：从配置读取告警队列主题
		true,           // 持久性：true表示队列数据持久化保存，MQ重启后数据不丢失
		false,          // 自动删除：false表示队列不会自动删除
		false,          // 排他性：false表示非排他队列（允许多消费者连接）
		false,          // 非阻塞：false表示同步声明队列，等待声明完成
		nil,            // 其他额外配置参数：无特殊配置
	)
	if err != nil {
		logrus.Fatalf("声明队列失败: %v", err)
		return
	}

	// 启动告警消息接收协程，异步处理MQ队列中的告警消息（避免阻塞当前启动流程）
	go RevAlertMq()
}
