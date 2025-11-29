package info

// File: honey_node/utils/info/system.go
// Description: 系统信息采集工具包，通过读取系统文件和执行系统命令获取Linux系统的发行版本、内核、架构及启动时间等信息

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SystemInfo 系统信息结构体
type SystemInfo struct {
	OSVersion    string `json:"osVersion,omitempty"`    // 操作系统发行版本
	Kernel       string `json:"kernel,omitempty"`       // 内核版本
	Architecture string `json:"architecture,omitempty"` // 系统架构
	BootTime     string `json:"bootTime,omitempty"`     // 系统启动时间
}

// GetSystemInfo 采集系统核心信息的入口函数
func GetSystemInfo() (SystemInfo, error) {
	info := SystemInfo{}
	var err error

	// 分步采集各系统信息，任一环节失败则返回错误
	info.OSVersion, err = getOSVersion()
	if err != nil {
		return info, fmt.Errorf("获取发行版本失败: %v", err)
	}

	info.Kernel, err = getKernelVersion()
	if err != nil {
		return info, fmt.Errorf("获取内核版本失败: %v", err)
	}

	info.Architecture, err = getArchitecture()
	if err != nil {
		return info, fmt.Errorf("获取系统架构失败: %v", err)
	}

	info.BootTime, err = getBootTime()
	if err != nil {
		return info, fmt.Errorf("获取系统启动时间失败: %v", err)
	}

	return info, nil
}

// getOSVersion 读取/etc/os-release文件获取操作系统发行版本信息
func getOSVersion() (string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				// 去除字段值两端的引号（单引号/双引号），获取纯净版本名称
				name := strings.Trim(parts[1], "\"'")
				return name, nil
			}
		}
	}

	return "", fmt.Errorf("未找到发行版本信息")
}

// getKernelVersion 通过执行uname -r命令获取内核版本
func getKernelVersion() (string, error) {
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getArchitecture 通过执行uname -m命令获取系统架构
func getArchitecture() (string, error) {
	cmd := exec.Command("uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getBootTime 读取/proc/stat文件中的btime字段获取系统启动时间
func getBootTime() (string, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime ") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				if btime, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					// 将Unix时间戳转换为人类可读的格式化时间字符串
					bootTime := time.Unix(btime, 0)
					return bootTime.Format("2006-01-02 15:04:05"), nil
				}
			}
		}
	}

	return "", fmt.Errorf("未找到系统启动时间信息")
}
