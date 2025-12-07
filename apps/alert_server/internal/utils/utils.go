package utils

// File: alert_server/utils/utils.go
// Description: 通用工具函数模块

// InList 检查指定元素是否存在于切片中（支持任意可比较类型）
func InList[T comparable](list []T, key T) bool {
	for _, t := range list {
		if t == key {
			return true
		}
	}
	return false
}

// Unique 移除切片中的重复元素，保持原有顺序
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(slice))

	for _, item := range slice {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}
