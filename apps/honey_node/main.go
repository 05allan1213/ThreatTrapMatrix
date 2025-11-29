package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/info"
	"honey_node/internal/utils/ip"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// 读取系统配置文件
	global.Config = core.ReadConfig()
	// 设置日志默认配置
	core.SetLogDefault()
	// 初始化全局日志实例
	global.Log = core.GetLogger()

	// 从配置获取gRPC管理服务地址
	addr := global.Config.System.GrpcManageAddr

	// 加载客户端证书和私钥（用于服务端验证客户端身份）
	cert, err := tls.LoadX509KeyPair("cert/client.crt", "cert/client.key")
	if err != nil {
		logrus.Fatalf("failed to load client key pair: %v", err)
	}

	// 加载CA根证书（用于验证服务端证书合法性）
	caCert, err := ioutil.ReadFile("cert/ca.crt")
	if err != nil {
		logrus.Fatalf("failed to read CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert) // 将CA证书加入证书池

	// 创建TLS配置（双向认证：客户端提供证书，服务端证书由CA验证）
	config := &tls.Config{
		Certificates: []tls.Certificate{cert}, // 客户端证书链
		RootCAs:      caCertPool,              // 信任的CA根证书池
	}

	// 将TLS配置封装为gRPC可用的凭证
	creds := credentials.NewTLS(config)

	// 建立加密gRPC连接
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		logrus.Fatalf(fmt.Sprintf("grpc connect addr [%s] 连接失败 %s", addr, err))
	}
	defer conn.Close() // 延迟关闭连接，确保资源释放

	// 初始化节点服务gRPC客户端实例
	client := node_rpc.NewNodeServiceClient(conn)

	// 采集节点基础信息
	// 获取指定网卡的IPv4地址和MAC地址
	_ip, mac, err := ip.GetNetworkInfo(global.Config.System.Network)
	if err != nil {
		logrus.Fatalln(err)
	}

	// 生成并持久化节点唯一标识（首次运行时）
	if global.Config.System.Uid == "" {
		global.Config.System.Uid = uuid.New().String()
		core.SetConfig() // 保存配置到文件
	}

	// 获取节点主机名
	hostname, err := os.Hostname()
	if err != nil {
		logrus.Fatalln(err)
	}

	// 获取节点系统信息
	systemInfo, err := info.GetSystemInfo()
	if err != nil {
		logrus.Fatalln(err)
	}

	// 调用管理端Register接口发送节点注册请求
	_, err = client.Register(context.Background(), &node_rpc.RegisterRequest{
		Ip:      _ip,                      // 节点ip
		Mac:     mac,                      // 节点mac
		NodeUid: global.Config.System.Uid, // 节点唯一标识
		Version: global.Version,           // 节点程序版本
		Commit:  global.Commit,            // 节点commit
		SystemInfo: &node_rpc.SystemInfoMessage{
			HostName:            hostname,
			DistributionVersion: systemInfo.OSVersion,
			CoreVersion:         systemInfo.Kernel,
			SystemType:          systemInfo.Architecture,
			StartTime:           systemInfo.BootTime,
		},
	})
	if err != nil {
		logrus.Fatalf("节点注册失败 %s", err)
		return
	}

	// 采集并上报节点资源信息
	// 获取当前工作目录作为节点数据路径
	nodePath, _ := os.Getwd()
	fmt.Println(nodePath)

	// 采集系统资源信息（CPU、内存、磁盘等）
	resourceInfo, err := info.GetResourceInfo(nodePath)
	if err != nil {
		logrus.Fatalf("节点资源信息获取失败 %s", err)
		return
	}

	// 上报资源信息到管理端
	_, err = client.NodeResource(context.Background(), &node_rpc.NodeResourceRequest{
		NodeUid: global.Config.System.Uid, // 节点唯一标识
		ResourceInfo: &node_rpc.ResourceMessage{ // 资源信息结构体映射
			CpuCount:              resourceInfo.CpuCount,
			CpuUseRate:            resourceInfo.CpuUseRate,
			MemTotal:              resourceInfo.MemTotal,
			MemUseRate:            resourceInfo.MemUseRate,
			DiskTotal:             resourceInfo.DiskTotal,
			DiskUseRate:           resourceInfo.DiskUseRate,
			NodePath:              resourceInfo.NodePath,
			NodeResourceOccupancy: resourceInfo.NodeResourceOccupancy,
		},
	})
	if err != nil {
		logrus.Fatalf("节点资源信息上报失败 %s", err)
		return
	}
}
