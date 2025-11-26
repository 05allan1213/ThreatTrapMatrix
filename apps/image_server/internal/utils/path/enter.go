package path

// File: image_server/utils/path.go
// Description: 提供路径相关工具方法，包含获取应用程序当前工作目录（根路径）的功能

import (
	"os"
)

// GetRootPath 获取应用程序当前工作目录路径
func GetRootPath() (path string) {
	// 获取当前工作目录
	path, err := os.Getwd()
	if err != nil {
		return ""
	}
	return path
}
