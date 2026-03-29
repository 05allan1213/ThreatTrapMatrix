package models

import "fmt"

const (
	// TaskTypeBatchDeploy 批量部署任务
	TaskTypeBatchDeploy int8 = 1
	// TaskTypeBatchUpdate 批量更新任务
	TaskTypeBatchUpdate int8 = 2
	// TaskTypeBatchRemove 批量删除任务
	TaskTypeBatchRemove int8 = 3
)

const (
	// TaskStatusRunning 任务执行中
	TaskStatusRunning int8 = 0
	// TaskStatusCompleted 任务执行完成
	TaskStatusCompleted int8 = 1
)

const (
	// TaskItemStagePending 本地动作尚未完成
	TaskItemStagePending int8 = 0
	// TaskItemStageLocalDone 本地动作已完成，但状态尚未成功回传
	TaskItemStageLocalDone int8 = 1
	// TaskItemStageReported 状态已经成功回传
	TaskItemStageReported int8 = 2
)

// TaskModel 任务模型
type TaskModel struct {
	Model
	TaskID                string                    `json:"taskID"`                                       // 任务ID
	Type                  int8                      `json:"type"`                                         // 任务类型 1 批量部署 2 批量更新 3 批量删除
	BatchDeployData       *BatchDeployRequest       `gorm:"serializer:json" json:"batchDeployData"`       // 批量部署参数
	DeployItems           []DeployTaskItemState     `gorm:"serializer:json" json:"deployItems"`           // 批量部署逐IP检查点
	BatchUpdateDeployData *BatchUpdateDeployRequest `gorm:"serializer:json" json:"batchUpdateDeployData"` // 批量更新参数
	UpdateItems           []UpdateTaskItemState     `gorm:"serializer:json" json:"updateItems"`           // 批量更新逐IP检查点
	BatchRemoveDeployData *BatchRemoveDeployRequest `gorm:"serializer:json" json:"batchRemoveDeployData"` // 批量删除参数
	RemoveItems           []RemoveTaskItemState     `gorm:"serializer:json" json:"removeItems"`           // 批量删除逐IP检查点
	Status                int8                      `json:"status"`                                       // 任务状态 0 运行中 1 运行完成
}

// DeployTaskItemState 批量部署逐IP检查点
type DeployTaskItemState struct {
	IP       string `json:"ip"`       // 诱捕IP
	Stage    int8   `json:"stage"`    // 检查点状态 0 待执行 1 本地完成未回传 2 已回传
	LinkName string `json:"linkName"` // 本地虚拟网卡名称
	Mac      string `json:"mac"`      // 诱捕IP的MAC地址
	ErrorMsg string `json:"errorMsg"` // 部署错误信息
	Manuf    string `json:"manuf"`    // 存活主机厂商信息
}

// UpdateTaskPortState 批量更新端口状态快照
type UpdateTaskPortState struct {
	Port     int    `json:"port"`     // 端口号
	ErrorMsg string `json:"errorMsg"` // 端口级错误信息
}

// UpdateTaskItemState 批量更新逐IP检查点
type UpdateTaskItemState struct {
	IP       string                `json:"ip"`       // 诱捕IP
	Stage    int8                  `json:"stage"`    // 检查点状态 0 待执行 1 本地完成未回传 2 已回传
	ErrorMsg string                `json:"errorMsg"` // IP级错误信息
	PortList []UpdateTaskPortState `json:"portList"` // 当前IP的端口状态快照
}

// RemoveTaskItemState 批量删除逐IP检查点
type RemoveTaskItemState struct {
	IP       string `json:"ip"`       // 诱捕IP
	LinkName string `json:"linkName"` // 虚拟网卡名称
	Stage    int8   `json:"stage"`    // 检查点状态 0 待执行 1 本地完成未回传 2 已回传
	ErrorMsg string `json:"errorMsg"` // 删除错误信息
}

// BatchDeployRequest MQ消费的批量部署请求结构体
type BatchDeployRequest struct {
	NetID   uint       `json:"netID"`   // 子网ID
	LogID   string     `json:"logID"`   // 日志ID
	Network string     `json:"network"` // 网卡名称
	TanIp   string     `json:"tanIp"`   // 探针IP
	IPList  []DeployIp `json:"ipList"`  // 待部署IP列表
}

// BatchUpdateDeployRequest MQ消费的批量更新部署请求结构体
type BatchUpdateDeployRequest struct {
	NetID    uint       `json:"netID"`    // 子网ID
	LogID    string     `json:"logID"`    // 日志ID
	IpList   []string   `json:"ipList"`   // 待更新IP列表
	PortList []PortInfo `json:"portList"` // 待更新端口列表
}

// BatchRemoveDeployRequest MQ消费的批量删除部署请求结构体
type BatchRemoveDeployRequest struct {
	NetID  uint             `json:"netID"`  // 子网ID
	LogID  string           `json:"logID"`  // 日志ID
	TanIp  string           `json:"tanIp"`  // 探针IP
	IPList []RemoveDeployIp `json:"ipList"` // 待删除IP列表
}

// RemoveDeployIp 单IP删除配置结构体
type RemoveDeployIp struct {
	Ip       string `json:"ip"`       // 待删除的IP地址
	LinkName string `json:"linkName"` // 待删除的IP对应的网络接口名称
}

// DeployIp 单IP部署配置结构体
type DeployIp struct {
	Ip       string     `json:"ip"`       // 待部署的诱捕IP地址
	Mask     int8       `json:"mask"`     // IP子网掩码
	PortList []PortInfo `json:"portList"` // 该IP关联的端口转发配置列表
}

// PortInfo 端口信息结构体
type PortInfo struct {
	IP       string `json:"ip"`       // 源ip
	Port     int    `json:"port"`     // 源端口
	DestIP   string `json:"destIP"`   // 目标ip
	DestPort int    `json:"destPort"` // 目标端口
}

// LocalAddr 本地监听地址
func (p PortInfo) LocalAddr() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

// TargetAddr 目标服务地址
func (p PortInfo) TargetAddr() string {
	return fmt.Sprintf("%s:%d", p.DestIP, p.DestPort)
}

// BuildDeployTaskItems 基于原始批量部署请求初始化逐IP检查点
func BuildDeployTaskItems(req *BatchDeployRequest) []DeployTaskItemState {
	if req == nil {
		return nil
	}

	items := make([]DeployTaskItemState, 0, len(req.IPList))
	for _, ip := range req.IPList {
		items = append(items, DeployTaskItemState{
			IP:    ip.Ip,
			Stage: TaskItemStagePending,
		})
	}
	return items
}

// BuildUpdateTaskItems 基于原始批量更新请求初始化逐IP检查点
func BuildUpdateTaskItems(req *BatchUpdateDeployRequest) []UpdateTaskItemState {
	if req == nil {
		return nil
	}

	items := make([]UpdateTaskItemState, 0, len(req.IpList))
	for _, ip := range req.IpList {
		items = append(items, UpdateTaskItemState{
			IP:    ip,
			Stage: TaskItemStagePending,
		})
	}
	return items
}

// BuildRemoveTaskItems 基于原始批量删除请求初始化逐IP检查点
func BuildRemoveTaskItems(req *BatchRemoveDeployRequest) []RemoveTaskItemState {
	if req == nil {
		return nil
	}

	items := make([]RemoveTaskItemState, 0, len(req.IPList))
	for _, ip := range req.IPList {
		items = append(items, RemoveTaskItemState{
			IP:       ip.Ip,
			LinkName: ip.LinkName,
			Stage:    TaskItemStagePending,
		})
	}
	return items
}

// EnsureTaskItems 兼容老任务数据，启动恢复时按原始请求补建检查点
func (m *TaskModel) EnsureTaskItems() bool {
	switch m.Type {
	case TaskTypeBatchDeploy:
		if len(m.DeployItems) == 0 && m.BatchDeployData != nil {
			m.DeployItems = BuildDeployTaskItems(m.BatchDeployData)
			return true
		}
	case TaskTypeBatchUpdate:
		if len(m.UpdateItems) == 0 && m.BatchUpdateDeployData != nil {
			m.UpdateItems = BuildUpdateTaskItems(m.BatchUpdateDeployData)
			return true
		}
	case TaskTypeBatchRemove:
		if len(m.RemoveItems) == 0 && m.BatchRemoveDeployData != nil {
			m.RemoveItems = BuildRemoveTaskItems(m.BatchRemoveDeployData)
			return true
		}
	}
	return false
}

// AllTaskItemsReported 判断当前任务是否已经完成所有逐IP状态回传
func (m *TaskModel) AllTaskItemsReported() bool {
	switch m.Type {
	case TaskTypeBatchDeploy:
		if len(m.DeployItems) == 0 {
			return false
		}
		for _, item := range m.DeployItems {
			if item.Stage != TaskItemStageReported {
				return false
			}
		}
		return true
	case TaskTypeBatchUpdate:
		if len(m.UpdateItems) == 0 {
			return false
		}
		for _, item := range m.UpdateItems {
			if item.Stage != TaskItemStageReported {
				return false
			}
		}
		return true
	case TaskTypeBatchRemove:
		if len(m.RemoveItems) == 0 {
			return false
		}
		for _, item := range m.RemoveItems {
			if item.Stage != TaskItemStageReported {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// FindDeployItem 查找批量部署任务中指定IP的检查点
func (m *TaskModel) FindDeployItem(ip string) *DeployTaskItemState {
	for i := range m.DeployItems {
		if m.DeployItems[i].IP == ip {
			return &m.DeployItems[i]
		}
	}
	return nil
}

// FindUpdateItem 查找批量更新任务中指定IP的检查点
func (m *TaskModel) FindUpdateItem(ip string) *UpdateTaskItemState {
	for i := range m.UpdateItems {
		if m.UpdateItems[i].IP == ip {
			return &m.UpdateItems[i]
		}
	}
	return nil
}

// FindRemoveItem 查找批量删除任务中指定IP的检查点
func (m *TaskModel) FindRemoveItem(ip string) *RemoveTaskItemState {
	for i := range m.RemoveItems {
		if m.RemoveItems[i].IP == ip {
			return &m.RemoveItems[i]
		}
	}
	return nil
}
