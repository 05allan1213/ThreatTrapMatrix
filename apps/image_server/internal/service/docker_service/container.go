package docker_service

// File: image_server/service/docker_service/container.go
// Description: Docker容器服务工具，基于Docker SDK实现容器创建、配置及启动，支持IP地址合法性校验

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"context"
	"net/netip"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// RunContainer 创建并启动Docker容器
func RunContainer(containerName, networkName, ip, image string) (containerID string, err error) {
	// 解析并校验IP地址合法性（防止无效IP配置导致容器创建失败）
	parsedIP, err := netip.ParseAddr(ip)
	if err != nil {
		return "", err
	}

	// 构建容器基础配置（指定运行镜像）
	containerConfig := &container.Config{
		Image: image,
	}

	// 构建容器主机配置（网络模式、自动删除策略）
	hostConfig := &container.HostConfig{
		AutoRemove:  false,                              // 容器退出后不自动删除
		NetworkMode: container.NetworkMode(networkName), // 关联指定Docker网络
	}

	// 构建容器网络配置（指定静态IP地址）
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {
				IPAMConfig: &network.EndpointIPAMConfig{
					IPv4Address: parsedIP, // 使用校验后的合法IP地址
				},
			},
		},
	}

	// 组装容器创建选项
	createOpts := client.ContainerCreateOptions{
		Name:             containerName,    // 容器名称（保证唯一性）
		Config:           containerConfig,  // 容器基础配置
		HostConfig:       hostConfig,       // 容器主机相关配置
		NetworkingConfig: networkingConfig, // 容器网络配置
	}

	// 调用Docker SDK创建容器（未启动状态）
	createResp, err := global.DockerClient.ContainerCreate(
		context.Background(),
		createOpts,
	)
	if err != nil {
		return
	}

	// 启动已创建的容器
	_, err = global.DockerClient.ContainerStart(context.Background(), createResp.ID, client.ContainerStartOptions{})
	if err != nil {
		return
	}

	// 截取容器完整ID的前12位作为短ID返回
	containerID = createResp.ID[:12]
	return
}
