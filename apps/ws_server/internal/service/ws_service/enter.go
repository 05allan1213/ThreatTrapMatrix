package ws_service

// File: ws_server/service/ws_service/enter.go
// Description: WebSocket连接管理与消息推送服务模块，基于sync.Map实现并发安全的WS连接存储，提供连接添加、移除及全量消息广播能力，支撑服务端向所有在线客户端推送消息

import (
	"sync"
	"ws_server/internal/core"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// wsStore WebSocket连接存储容器
var wsStore = sync.Map{}

// AddWs 添加WebSocket连接到存储容器
func AddWs(conn *websocket.Conn) {
	wsStore.Store(conn.RemoteAddr(), conn)
}

// RemoveWs 从存储容器移除指定地址的WebSocket连接
func RemoveWs(addr string) {
	wsStore.Delete(addr)
}

// SendMsg 向所有在线的WebSocket客户端广播消息
func SendMsg(msg []byte, logID string) {
	// 初始化带日志追踪ID的日志实例
	log := core.GetLogger().WithField("logID", logID)
	var count int // 统计推送成功的客户端数量

	// 遍历所有在线WS连接执行消息推送
	wsStore.Range(func(key, value any) bool {
		// 类型断言转换为WebSocket连接实例
		conn := value.(*websocket.Conn)
		// 向客户端推送文本类型消息
		err := conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			// 单连接推送失败：记录结构化错误日志
			log.WithFields(map[string]interface{}{
				"error": err.Error(),
				"addr":  key,
			}).Error("消息推送失败")
			// 兼容原有logrus直接输出的日志
			logrus.Errorf("消息推送失败 %s", err)
			return true // 返回true继续遍历下一个连接
		}

		// 单连接推送成功：记录结构化成功日志
		log.WithFields(map[string]interface{}{
			"addr": key,
		}).Infof("消息推送成功")
		count++ // 累加成功计数
		// 兼容原有logrus直接输出的日志
		logrus.Infof("消息推送成功 %s", string(msg))
		return true // 返回true继续遍历下一个连接
	})

	// 推送完成：记录全量推送统计日志
	log.WithField("ws_count", count).Info("消息推送完成")
}
