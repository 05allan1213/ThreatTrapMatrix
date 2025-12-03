package mq_service

// File: alert_server/service/mq_service/rev_alert_mq.go
// Description: MQ告警消息消费模块，负责监听告警队列、解析消息、白名单过滤及虚拟服务信息关联

import (
	"alert_server/internal/es_models"
	"alert_server/internal/global"
	"alert_server/internal/models"
	"context"
	"encoding/json"
	"log"

	"github.com/sirupsen/logrus"
)

// RevAlertMq 消费MQ告警队列消息，核心流程：监听队列→解析消息→白名单过滤→关联虚拟服务信息
func RevAlertMq() {
	cfg := global.Config.Alert
	// 注册MQ消费者，监听指定告警队列
	msgs, err := global.Queue.Consume(
		cfg.AlertTopic, // 消费队列名称：从配置读取告警队列主题
		"",             // 消费者标识：空字符串表示使用默认标识
		true,           // 自动确认：消息处理后自动向MQ发送ACK确认
		false,          // 排他性：false表示非排他消费（允许多消费者同时消费）
		false,          // 非本地：false表示接受非本地队列的消息
		false,          // 非阻塞：false表示同步注册消费者，等待注册完成
		nil,            // 其他额外配置参数：无特殊配置
	)
	if err != nil {
		log.Fatalf("无法注册消费者: %v", err)
	}

	// 循环监听队列消息，持续消费新增告警
	for d := range msgs {
		// 将消息体JSON解析为ES告警数据模型
		var data es_models.AlertModel
		err = json.Unmarshal(d.Body, &data)
		if err != nil {
			logrus.Errorf("消息格式解析失败 %s %s", err, d.Body)
			continue // 解析失败跳过当前消息，继续消费下一条
		}

		// 打印告警核心信息：规则描述、源IP、目标IP:端口
		logrus.Infof("%s %s => %s:%d", data.Signature, data.SrcIp, data.DestIp, data.DestPort)

		// 白名单过滤：检查攻击源IP是否在白名单中，在则跳过后续处理
		var whiteModel models.WhiteIPModel
		global.DB.Find(&whiteModel, "ip = ?", data.SrcIp)
		if whiteModel.ID != 0 {
			logrus.Warnf("告警消息 在白名单中（源IP：%s），跳过处理", data.SrcIp)
			continue
		}

		// 关联虚拟服务信息：查询目标IP和端口对应的虚拟服务配置
		var hpModel models.HoneyPortModel
		global.DB.Preload("ServiceModel").Find(&hpModel, "ip = ? and port = ?", data.DestIp, data.DestPort)
		if hpModel.ID != 0 {
			// 补充虚拟服务ID和服务名称到告警数据中
			data.ServiceID = hpModel.ServiceID
			data.ServiceName = hpModel.ServiceModel.Title
		}

		// es告警信息入库
		response, err1 := global.ES.Index().Index(data.Index()).BodyJson(data).Do(context.Background())
		if err1 != nil {
			logrus.Errorf("告警消息入库失败 %s %s", err1, d.Body)
			continue
		}
		logrus.Infof("告警消息入库成功 %s", response.Id)
	}
}
