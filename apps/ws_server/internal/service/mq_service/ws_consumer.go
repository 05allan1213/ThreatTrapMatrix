package mq_service

import (
	"ws_server/internal/service/ws_service"
)

// wsConsumer 通用MQ消费者处理函数，用于处理WebSocket消息
func wsConsumer(req []byte) {
	ws_service.SendMsg(req)
}
