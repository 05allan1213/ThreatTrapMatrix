package ws_service

// File: ws_server/service/ws_service/enter.go
// Description: WebSocket服务模块，封装WebSocket连接全生命周期管理、心跳状态跟踪、消息广播推送等核心功能，提供并发安全的连接操作能力

import (
	"errors"
	"sync"
	"time"
	"ws_server/internal/core"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WsStore WebSocket连接存储容器
var WsStore = sync.Map{}

// WSConn WebSocket连接封装结构体
type WSConn struct {
	conn       *websocket.Conn // 原生WebSocket连接实例
	lastActive time.Time       // 最后活跃时间（用于心跳检测）
	isClosing  bool            // 连接关闭状态标记，避免重复关闭
	mu         sync.Mutex      // 互斥锁，保证连接操作的并发安全
}

// AddWs 添加新的WebSocket连接到存储容器
func AddWs(conn *websocket.Conn) {
	wsConn := &WSConn{
		conn:       conn,
		lastActive: time.Now(), // 初始化最后活跃时间为当前时间
	}

	// 设置Pong处理器：接收客户端Pong帧时更新最后活跃时间，用于心跳检测判断连接存活状态
	wsConn.conn.SetPongHandler(func(string) error {
		wsConn.mu.Lock()
		wsConn.lastActive = time.Now()
		wsConn.mu.Unlock()
		return nil
	})

	// 将封装后的连接存入sync.Map，key为客户端远程地址
	WsStore.Store(conn.RemoteAddr().String(), wsConn)
	logrus.Infof("新连接添加: %s, 当前连接数: %d", conn.RemoteAddr(), getConnectionCount())
}

// getConnectionCount 统计当前有效WebSocket连接数
func getConnectionCount() int {
	count := 0
	// 遍历sync.Map，每遍历一个元素计数+1
	WsStore.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// RemoveWs 从存储容器移除并关闭指定地址的WebSocket连接
func RemoveWs(addr string) {
	// 加载并删除指定地址的连接，避免重复操作
	if value, ok := WsStore.LoadAndDelete(addr); ok {
		wsConn := value.(*WSConn)
		wsConn.close() // 安全关闭连接
		logrus.Infof("连接已移除: %s, 当前连接数: %d", addr, getConnectionCount())
	}
}

// isClosed 并发安全检查连接是否已关闭
func (c *WSConn) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock() // 方法结束自动解锁，保证锁释放
	return c.isClosing
}

// close 安全关闭WebSocket连接
func (c *WSConn) close() {
	c.mu.Lock()
	// 若连接已标记为关闭，直接解锁返回，避免重复操作
	if c.isClosing {
		c.mu.Unlock()
		return
	}
	// 标记连接为关闭状态
	c.isClosing = true
	c.mu.Unlock()

	// 发送正常关闭帧，告知客户端连接即将关闭
	_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	// 延迟100ms，给客户端足够时间处理关闭帧
	time.Sleep(100 * time.Millisecond)
	// 关闭原生连接
	_ = c.conn.Close()
}

// SendMsg 广播消息到所有有效WebSocket连接
func SendMsg(msg []byte, logID string) {
	// 获取带logID的日志实例，便于追踪单次消息推送的全流程
	log := core.GetLogger().WithField("logID", logID)
	var count int // 记录成功推送的连接数

	// 遍历所有WebSocket连接，逐个推送消息
	WsStore.Range(func(key, value any) bool {
		addr := key.(string)
		wsConn := value.(*WSConn)

		// 检查连接是否已关闭，若已关闭则移除并继续下一个连接
		if wsConn.isClosed() {
			RemoveWs(addr)
			return true
		}

		// 发送消息时加锁，避免并发写连接导致的错误
		wsConn.mu.Lock()
		err := wsConn.conn.WriteMessage(websocket.TextMessage, msg)
		wsConn.mu.Unlock()

		if err != nil {
			// 分类处理消息推送错误，针对性记录日志并清理无效连接
			if errors.Is(err, websocket.ErrCloseSent) {
				log.WithFields(map[string]interface{}{
					"error": "连接已关闭",
					"addr":  addr,
				}).Warn("消息推送失败")
				RemoveWs(addr)
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.WithFields(map[string]interface{}{
					"error": err.Error(),
					"addr":  addr,
				}).Error("连接异常关闭")
				RemoveWs(addr)
			} else {
				log.WithFields(map[string]interface{}{
					"error": err.Error(),
					"addr":  addr,
				}).Error("消息推送失败")
			}
			return true
		}

		// 推送成功，记录日志并计数
		log.WithFields(map[string]interface{}{
			"addr": addr,
		}).Infof("消息推送成功")
		count++
		return true
	})

	// 记录本次消息推送的总体结果
	log.WithField("ws_count", count).Info("消息推送完成")
}
