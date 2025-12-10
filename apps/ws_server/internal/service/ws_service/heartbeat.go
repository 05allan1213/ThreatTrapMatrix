package ws_service

// File: ws_server/service/ws_service/heartbeat.go
// Description: WebSocket服务模块，提供WebSocket连接心跳检测功能，检测超时未活动连接并自动清理，保障连接有效性与服务稳定性

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// HeartbeatChecker WebSocket连接心跳检测核心函数
func HeartbeatChecker() {
	// 存储待清理的超时连接地址，避免遍历过程中直接修改WsStore引发并发问题
	var timeoutConnections []string

	// 第一阶段：遍历所有连接，检查活动状态并标记超时/异常连接
	WsStore.Range(func(key, value any) bool {
		addr := key.(string)
		wsConn := value.(*WSConn)

		wsConn.mu.Lock() // 加锁保证连接状态读取/操作的并发安全
		// 判定连接是否超时：最后活跃时间距当前超过60秒
		if time.Since(wsConn.lastActive) > 60*time.Second {
			timeoutConnections = append(timeoutConnections, addr)
		} else {
			// 连接未超时，发送Ping帧检测连接可用性（触发客户端回复Pong帧更新lastActive）
			err := wsConn.conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				logrus.Errorf("发送心跳失败: %s, %v", addr, err)
				timeoutConnections = append(timeoutConnections, addr)
			}
		}
		wsConn.mu.Unlock() // 解锁，释放连接操作权限
		return true        // 继续遍历下一个连接
	})

	// 第二阶段：批量清理标记的超时/异常连接
	for _, addr := range timeoutConnections {
		logrus.Warnf("连接超时，自动关闭: %s", addr)
		RemoveWs(addr) // 移除并安全关闭连接
	}

	// 记录心跳检测完成日志，输出当前剩余有效连接数
	logrus.Infof("心跳检测完成，当前连接数: %d", getConnectionCount())
}
