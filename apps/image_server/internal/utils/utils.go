package utils

// File: image_server/utils/utils.go
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
