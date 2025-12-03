package core

// File: alert_server/core/mq.go
// Description: RabbitMQ连接初始化模块，支持SSL/TLS加密连接与普通连接，创建并返回MQ通道实例

import (
	"crypto/tls"
	"crypto/x509"
	"alert_server/internal/global"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// InitMQ 初始化RabbitMQ连接并创建通道
func InitMQ() *amqp.Channel {
	cfg := global.Config.MQ // 获取全局MQ配置
	var conn *amqp.Connection
	var err error

	// 根据配置选择SSL/TLS连接或普通连接
	if cfg.Ssl {
		// 1. 加载客户端证书和私钥（用于双向SSL认证）
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertificate, cfg.ClientKey)
		if err != nil {
			logrus.Fatalf("加载客户端证书失败: %v", err)
		}

		// 2. 加载CA根证书（用于验证服务端证书合法性）
		caCert, err := os.ReadFile(cfg.CaCertificate)
		if err != nil {
			logrus.Fatalf("读取CA证书失败: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert) // 将CA证书加入信任池

		// 3. 配置TLS连接参数
		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert}, // 客户端证书链（双向认证）
			RootCAs:            caCertPool,              // 信任的CA根证书池
			InsecureSkipVerify: false,                   // 禁止跳过服务端证书验证（必须验证）
		}
		// 建立TLS加密连接
		conn, err = amqp.DialTLS(cfg.Addr(), tlsConfig)
	} else {
		// 建立普通TCP连接
		conn, err = amqp.Dial(cfg.Addr())
	}

	// 连接失败时终止程序
	if err != nil {
		logrus.Fatalf("无法连接到 RabbitMQ: %v", err)
	}

	// 创建MQ通道（Channel），用于消息收发
	ch, err := conn.Channel()
	if err != nil {
		logrus.Fatalf("无法打开通道: %v", err)
	}

	return ch
}
