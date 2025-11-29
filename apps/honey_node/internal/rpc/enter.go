package rpc

// File: honey_node/rpc/enter.go
// Description: gRPC连接管理工具，封装基于TLS双向认证的gRPC连接创建逻辑，提供安全的连接建立入口

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GetConn 创建基于TLS双向认证的gRPC客户端连接
func GetConn(addr string) (conn *grpc.ClientConn) {
	// 客户端证书加载（用于服务端验证客户端身份）
	cert, err := tls.LoadX509KeyPair("cert/client.crt", "cert/client.key")
	if err != nil {
		logrus.Fatalf("failed to load client key pair: %v", err)
	}

	// CA根证书加载（用于客户端验证服务端证书合法性）
	caCert, err := ioutil.ReadFile("cert/ca.crt")
	if err != nil {
		logrus.Fatalf("failed to read CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	// 将CA证书解析并加入信任池，用于验证服务端证书是否由可信CA签发
	caCertPool.AppendCertsFromPEM(caCert)

	// TLS双向认证配置
	config := &tls.Config{
		Certificates: []tls.Certificate{cert}, // 客户端证书链（包含公钥+私钥）
		RootCAs:      caCertPool,              // 信任的CA根证书池（用于服务端证书验证）
	}

	// 将TLS配置转换为gRPC可识别的传输凭证
	creds := credentials.NewTLS(config)

	// 建立gRPC连接
	conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		logrus.Fatalf(fmt.Sprintf("grpc connect addr [%s] 连接失败 %s", addr, err))
	}
	return
}
