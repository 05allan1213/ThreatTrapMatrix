package mq_service

import (
	"encoding/json"
	"ws_server/internal/core"
	"ws_server/internal/service/ws_service"
)

// wsConsumer 通用MQ消费者处理函数，用于处理WebSocket消息
func wsConsumer(req wsMsgType) {
	log := core.GetLogger().WithField("logID", req.LogID)
	log.WithField("ws_data", req).Infof("实时推送消息")
	byteData, _ := json.Marshal(req)
	ws_service.SendMsg(byteData, req.LogID)
}
