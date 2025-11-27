package vs_net_service

// File: image_server/service/vs_net_service/enter.go
// Description: 虚拟子网初始化服务，负责检查并初始化Docker虚拟网络，确保网络配置与系统配置一致

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"context"

	"github.com/docker/docker/api/types/network"
	"github.com/sirupsen/logrus"
)

// Run 初始化虚拟子网服务
func Run() {
	// 获取全局配置中的虚拟子网配置，并设置默认值（防止配置缺失）
	cfg := global.Config.VsNet
	if cfg.Name == "" {
		cfg.Name = "honey-hy" // 默认网络名称
	}
	if cfg.Net == "" {
		cfg.Net = "10.2.0.0/24" // 默认子网段
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "hy-" // 默认容器名称前缀
	}

	// 获取Docker引擎中所有网络列表
	networks, err := global.DockerClient.NetworkList(context.Background(), network.ListOptions{})
	if err != nil {
		logrus.Fatalf("获取虚拟网络列表失败: %v", err)
	}

	// 查找配置中指定的网络是否已存在
	var found bool
	var existingNetwork network.Summary
	for _, network := range networks {
		if network.Name == cfg.Name {
			found = true
			existingNetwork = network
			break
		}
	}

	// 网络不存在时，根据配置创建新的Docker虚拟网络
	if !found {
		// 配置网络IPAM（IP地址管理）参数
		ipam := network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet: cfg.Net, // 使用配置的子网段
				},
			},
		}
		// 创建Docker网络
		_, err := global.DockerClient.NetworkCreate(context.Background(), cfg.Name, network.CreateOptions{
			IPAM: &ipam,
		})
		if err != nil {
			logrus.Fatalf("创建网络失败: %v", err)
		}
		logrus.Printf("成功创建网络 %s，子网为 %s", cfg.Name, cfg.Net)
		return
	}

	// 网络已存在，校验子网配置是否与系统配置一致
	if len(existingNetwork.IPAM.Config) > 0 && existingNetwork.IPAM.Config[0].Subnet != cfg.Net {
		logrus.Warnf("警告: 网络 %s 存在，但子网不匹配。现有子网: %s，配置子网: %s",
			cfg.Name, existingNetwork.IPAM.Config[0].Subnet, cfg.Net)
		logrus.Fatalf("请排查网络配置问题")
		return
	}

	logrus.Infof("网络 %s 存在且子网匹配: %s", cfg.Name, cfg.Net)
}
