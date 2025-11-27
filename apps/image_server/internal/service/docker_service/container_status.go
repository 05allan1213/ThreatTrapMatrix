package docker_service

// File: image_server/service/docker_service/container_status.go
// Description: Docker容器信息查询服务，提供容器列表获取、容器状态查询等功能

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// ListAllContainers 获取Docker引擎中所有容器的状态信息（包括运行和停止的容器）
func ListAllContainers() ([]container.Summary, error) {
	// 获取所有容器列表（All=true表示包含停止的容器）
	containers, err := global.DockerClient.ContainerList(context.Background(), container.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %v", err)
	}

	return containers, nil
}

// GetContainerStatus 根据容器名称查询指定容器的状态信息
func GetContainerStatus(containerName string) (container.Summary, error) {
	// 创建过滤器：按容器名称筛选
	filter := filters.NewArgs()
	filter.Add("name", containerName)

	// 查询匹配名称的容器列表（All=true表示包含停止的容器）
	containers, err := global.DockerClient.ContainerList(context.Background(), container.ListOptions{
		Filters: filter,
		All:     true,
	})
	if err != nil {
		return container.Summary{}, fmt.Errorf("获取容器列表失败: %v", err)
	}

	// 检查是否找到匹配的容器
	if len(containers) == 0 {
		return container.Summary{}, fmt.Errorf("未找到名为 %s 的容器", containerName)
	}

	// 返回第一个匹配的容器（Docker容器名称具有唯一性）
	return containers[0], nil
}
