package cron_service

// File: ws_server/service/cron_service/heartbeat_checker.go
// Description: 定时任务服务模块，封装节点心跳检测定时任务的入口函数，调用WebSocket服务的心跳检测逻辑完成节点状态检测

import "ws_server/internal/service/ws_service"

// HeartbeatChecker 节点心跳检测定时任务入口函数
func HeartbeatChecker() {
	// 调用WebSocket服务的心跳检测函数，执行节点心跳状态检测
	ws_service.HeartbeatChecker()
}
