package ws_service

// File: ws_server/service/ws_service/enter.go
// Description: WebSocket连接管理与消息推送服务模块，基于sync.Map实现并发安全的WS连接存储，提供连接添加、移除及全量消息广播能力，支撑服务端向所有在线客户端推送消息

import (
	"sync"

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
func SendMsg(msg []byte) {
	// 遍历所有在线WS连接，执行消息推送
	wsStore.Range(func(key, value any) bool {
		// 类型断言转换为WebSocket连接实例
		conn := value.(*websocket.Conn)
		// 向客户端推送文本消息
		err := conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			logrus.Errorf("消息推送失败 %s", err)
			return true // 返回true继续遍历下一个连接
		}
		// 推送成功记录日志
		logrus.Infof("消息推送成功 %s", string(msg))
		return true // 返回true继续遍历下一个连接
	})
}
