package docker_service

import (
	"context"
	"fmt"
	"image_server/internal/global"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// RemoveContainerByName 根据容器名删除容器（容器不存在视为成功）
func RemoveContainerByName(containerName string) error {
	if containerName == "" {
		return nil
	}
	filter := filters.NewArgs()
	filter.Add("name", containerName)

	containers, err := global.DockerClient.ContainerList(context.Background(), container.ListOptions{
		Filters: filter,
		All:     true,
	})
	if err != nil {
		return fmt.Errorf("查询容器失败: %w", err)
	}

	// 精确匹配容器名（避免前缀误伤）
	var targetID string
	for _, item := range containers {
		for _, name := range item.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				targetID = item.ID
				break
			}
		}
		if targetID != "" {
			break
		}
	}
	// 未找到容器，视为已删除
	if targetID == "" {
		return nil
	}

	// 强制删除，保证幂等
	if err := global.DockerClient.ContainerRemove(context.Background(), targetID, container.RemoveOptions{
		Force: true,
	}); err != nil {
		if isContainerNotFound(err) {
			return nil
		}
		return fmt.Errorf("删除容器失败: %w", err)
	}
	return nil
}

// isContainerNotFound 判断删除时的“容器不存在”错误
func isContainerNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such container") || strings.Contains(msg, "not found")
}
