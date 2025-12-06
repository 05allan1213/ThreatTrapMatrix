package api

// File: ws_server/api/ws.go
// Description: WebSocket通信API接口

import (
	"net/http"
	"ws_server/internal/service/ws_service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// UP WebSocket连接升级器实例
// 配置读写缓冲区大小，关闭跨域Origin检查（适配测试/内网场景），用于将HTTP连接升级为WS连接
var UP = websocket.Upgrader{
	ReadBufferSize:  1024, // 读缓冲区大小1024字节
	WriteBufferSize: 1024, // 写缓冲区大小1024字节
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有Origin跨域连接（生产环境需根据实际场景限制）
	},
}

// WsView WebSocket通信接口处理函数
// 完成HTTP到WS的连接升级，建立长连接后循环读取客户端消息，返回echo格式的响应，连接异常时关闭并记录日志
func (Api) WsView(c *gin.Context) {
	// 1. 将HTTP连接升级为WebSocket连接
	conn, err := UP.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("ws升级失败 %s", err)
		return
	}
	addr := conn.RemoteAddr()
	logrus.Infof("客户端连接成功 %s", addr)
	ws_service.AddWs(conn)

	// 2. 循环读取客户端消息，维持长连接
	for {
		// 读取客户端消息：t为消息类型（文本/二进制等），p为消息内容，err为读取错误
		_, _, err := conn.ReadMessage()
		if err != nil {
			break // 读取失败（如客户端断开），退出循环
		}
	}

	// 3. 连接清理：关闭WS连接，记录断开日志
	logrus.Infof("客户端断开连接 %s", addr)
	ws_service.RemoveWs(addr.String())
}
