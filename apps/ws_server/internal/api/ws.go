package api

// File: ws_server/api/ws.go
// Description: WebSocket通信API接口

import (
	"fmt"
	"net/http"

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
	logrus.Infof("客户端连接成功 %s", conn.RemoteAddr())

	// 2. 循环读取客户端消息，维持长连接
	for {
		// 读取客户端消息：t为消息类型（文本/二进制等），p为消息内容，err为读取错误
		t, p, err := conn.ReadMessage()
		if err != nil {
			break // 读取失败（如客户端断开），退出循环
		}
		// 回复echo消息：将客户端消息包装后以文本类型返回
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("你说的是：%s吗？", string(p))))
		fmt.Println(t, string(p)) // 控制台打印消息类型和内容（调试用）
	}

	// 3. 连接清理：关闭WS连接，记录断开日志
	defer conn.Close()                                   // 延迟关闭连接，确保最终执行
	logrus.Infof("客户端断开连接 %s", conn.RemoteAddr()) // 修正原日志文案，准确描述连接状态
}
