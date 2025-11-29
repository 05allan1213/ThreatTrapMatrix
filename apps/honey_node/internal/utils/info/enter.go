package info

// File: honey_node/utils/info/enter.go
// Description: 系统资源信息采集工具包，基于gopsutil库获取CPU、内存、磁盘等资源使用情况

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// ResourceMessage 系统资源信息结构体
type ResourceMessage struct {
	CpuCount              int64   `json:"cpuCount,omitempty"`              // CPU内核数（逻辑核心）
	CpuUseRate            float32 `json:"cpuUseRate,omitempty"`            // CPU使用率（百分比）
	MemTotal              int64   `json:"memTotal,omitempty"`              // 内存容量（字节）
	MemUseRate            float32 `json:"memUseRate,omitempty"`            // 内存使用率（百分比）
	DiskTotal             int64   `json:"diskTotal,omitempty"`             // 磁盘容量（字节）
	DiskUseRate           float32 `json:"diskUseRate,omitempty"`           // 磁盘使用率（百分比）
	NodePath              string  `json:"nodePath,omitempty"`              // 节点部署目录
	NodeResourceOccupancy int64   `json:"nodeResourceOccupancy,omitempty"` // 节点部署目录资源占用（字节）
}

// GetResourceInfo 采集系统资源信息
func GetResourceInfo(nodePath string) (*ResourceMessage, error) {
	// 获取CPU逻辑核心数（true表示包含超线程）
	cpuCount, err := cpu.Counts(true)
	if err != nil {
		return nil, err
	}

	// 获取CPU整体使用率（采样间隔1秒，false表示获取整体CPU使用率而非单个核心）
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, err
	}

	// 获取系统虚拟内存信息（总内存、已用内存、使用率等）
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	// 获取节点指定路径的磁盘使用情况（用于统计节点数据占用）
	nodeDiskInfo, err := disk.Usage(nodePath)
	if err != nil {
		return nil, err
	}

	// 获取系统根目录的磁盘使用情况（用于统计整体磁盘状态）
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}

	// 组装资源信息结构体并返回
	message := &ResourceMessage{
		CpuCount:              int64(cpuCount),
		CpuUseRate:            float32(cpuPercent[0]), // 取第一个元素（整体CPU使用率）
		MemTotal:              int64(memInfo.Total),
		MemUseRate:            float32(memInfo.UsedPercent),
		DiskTotal:             int64(diskInfo.Total),
		DiskUseRate:           float32(diskInfo.UsedPercent),
		NodePath:              nodePath,
		NodeResourceOccupancy: int64(nodeDiskInfo.Used),
	}

	return message, nil
}
