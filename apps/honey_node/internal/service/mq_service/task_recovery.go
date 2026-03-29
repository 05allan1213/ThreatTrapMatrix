package mq_service

// File: honey_node/service/mq_service/task_recovery.go
// Description: 批量任务恢复辅助模块，统一实现逐IP执行、检查点推进、重启补回调以及本地差集恢复逻辑

import (
	"fmt"
	"honey_node/internal/core"
	"honey_node/internal/global"
	"honey_node/internal/models"
	"honey_node/internal/service/ip_service"
	"honey_node/internal/service/port_service"
	"honey_node/internal/service/task_checkpoint"
	info2 "honey_node/internal/utils/info"
	"honey_node/internal/utils/random"
	"net"
	"strings"

	"github.com/j-keck/arping"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ResumeDeployTask 恢复未完成的批量部署任务
func ResumeDeployTask(taskID string) error {
	task, err := task_checkpoint.GetTask(taskID)
	if err != nil {
		return err
	}
	if task.BatchDeployData == nil {
		return task_checkpoint.BuildMissingTaskDataError(taskID)
	}

	req := *task.BatchDeployData
	log := core.GetLogger().WithField("logID", req.LogID).WithField("task_id", taskID)
	log.Info("恢复未完成的批量部署任务")

	for _, item := range task.DeployItems {
		switch item.Stage {
		case models.TaskItemStageReported:
			continue
		case models.TaskItemStageLocalDone:
			if err := resendDeployTaskItem(taskID, req, item); err != nil {
				log.WithError(err).Error("补发部署状态失败")
			}
		default:
			info, ok := findDeployInfo(req, item.IP)
			if !ok {
				// 原始请求缺项属于异常数据，直接补一条失败状态，避免中心侧永远等不到回调
				err = syncDeployTaskResult(taskID, DeployStatusRequest{
					NetID:    req.NetID,
					IP:       item.IP,
					LogID:    req.LogID,
					ErrorMsg: "恢复任务缺少原始部署数据",
				})
				if err != nil {
					log.WithError(err).Error("补发异常部署状态失败")
				}
				continue
			}
			if err := processDeployTaskItem(taskID, req, info); err != nil {
				log.WithError(err).WithField("ip", info.Ip).Error("恢复部署IP失败")
			}
		}
	}

	// 批量部署除了补回调，还要把缺失的端口规则补齐
	reconcileDeployTaskPorts(req, log)
	return task_checkpoint.CompleteTaskIfReported(taskID)
}

// ResumeUpdateTask 恢复未完成的批量更新任务
func ResumeUpdateTask(taskID string) error {
	task, err := task_checkpoint.GetTask(taskID)
	if err != nil {
		return err
	}
	if task.BatchUpdateDeployData == nil {
		return task_checkpoint.BuildMissingTaskDataError(taskID)
	}

	req := *task.BatchUpdateDeployData
	log := core.GetLogger().WithField("logID", req.LogID).WithField("task_id", taskID)
	log.Info("恢复未完成的批量更新任务")

	portMap := buildUpdatePortMap(req)
	for _, item := range task.UpdateItems {
		switch item.Stage {
		case models.TaskItemStageReported:
			continue
		case models.TaskItemStageLocalDone:
			if err := resendUpdateTaskItem(taskID, req, item); err != nil {
				log.WithError(err).Error("补发更新状态失败")
			}
		default:
			desiredPorts := portMap[item.IP]
			// 本地规则已经和目标一致时，不重复改规则，只补发缺失状态
			if updateTaskItemMatchesDesired(item.IP, desiredPorts) {
				err = syncUpdateTaskResult(taskID, UpdateDeployStatusRequest{
					NetID:    req.NetID,
					IP:       item.IP,
					LogID:    req.LogID,
					ErrorMsg: "",
					PortList: buildUpdatePortStatus(desiredPorts),
				})
				if err != nil {
					log.WithError(err).WithField("ip", item.IP).Error("补发更新成功状态失败")
				}
				continue
			}
			if err := processUpdateTaskItem(taskID, req, item.IP, desiredPorts, log); err != nil {
				log.WithError(err).WithField("ip", item.IP).Error("恢复更新IP失败")
			}
		}
	}

	return task_checkpoint.CompleteTaskIfReported(taskID)
}

// ResumeRemoveTask 恢复未完成的批量删除任务
func ResumeRemoveTask(taskID string) error {
	task, err := task_checkpoint.GetTask(taskID)
	if err != nil {
		return err
	}
	if task.BatchRemoveDeployData == nil {
		return task_checkpoint.BuildMissingTaskDataError(taskID)
	}

	req := *task.BatchRemoveDeployData
	log := core.GetLogger().WithField("logID", req.LogID).WithField("task_id", taskID)
	log.Info("恢复未完成的批量删除任务")

	for _, item := range task.RemoveItems {
		switch item.Stage {
		case models.TaskItemStageReported:
			continue
		case models.TaskItemStageLocalDone:
			if err := resendRemoveTaskItem(taskID, req, item); err != nil {
				log.WithError(err).Error("补发删除状态失败")
			}
		default:
			// 本地已经删干净时，不重复删除，只补发缺失状态
			if removeTaskItemCompleted(item) {
				err = syncRemoveTaskResult(taskID, RemoveDeployStatusRequest{
					NetID:    req.NetID,
					IP:       item.IP,
					LogID:    req.LogID,
					ErrorMsg: "",
				})
				if err != nil {
					log.WithError(err).WithField("ip", item.IP).Error("补发删除成功状态失败")
				}
				continue
			}
			if err := processRemoveTaskItem(taskID, req, item.IP, item.LinkName, log); err != nil {
				log.WithError(err).WithField("ip", item.IP).Error("恢复删除IP失败")
			}
		}
	}

	return task_checkpoint.CompleteTaskIfReported(taskID)
}

// processDeployTaskItem 执行单个部署项，并按检查点状态推进回调
func processDeployTaskItem(taskID string, req models.BatchDeployRequest, info models.DeployIp) error {
	res := DeployStatusRequest{
		NetID: req.NetID,
		IP:    info.Ip,
		LogID: req.LogID,
	}

	// 探针IP不需要重复创建，只补MAC并上报成功
	if info.Ip == req.TanIp {
		res.Mac, _ = ip_service.GetMACAddress(req.Network)
		return syncDeployTaskResult(taskID, res)
	}

	// 本地SQLite已经有记录时，说明该IP已部署，直接补回调即可
	var ipModel models.IpModel
	global.DB.Find(&ipModel, "network = ? and ip = ?", req.Network, info.Ip)
	if ipModel.ID != 0 {
		res.Mac = ipModel.Mac
		res.LinkName = ipModel.LinkName
		return syncDeployTaskResult(taskID, res)
	}

	// 崩溃可能发生在建完网卡、未写SQLite之间，这里额外兜底真实网卡是否已经存在
	if linkName, mac, ok := findHoneyInterface(info.Ip); ok {
		res.Mac = mac
		res.LinkName = linkName
		global.DB.Where(models.IpModel{Network: req.Network, Ip: info.Ip}).FirstOrCreate(&models.IpModel{
			Ip:       info.Ip,
			Mask:     info.Mask,
			LinkName: linkName,
			Network:  req.Network,
			Mac:      mac,
		})
		return syncDeployTaskResult(taskID, res)
	}

	// 非诱捕虚拟接口上已存在同IP时，仍按冲突失败处理
	if info2.FindLocalIp(info.Ip) {
		res.ErrorMsg = fmt.Sprintf("当前ip存在与本地ip冲突")
		return syncDeployTaskResult(taskID, res)
	}

	// ARP检测到存活主机时，仍按原有语义上报失败
	arpMac, _, err := arping.PingOverIfaceByName(net.ParseIP(info.Ip), req.Network)
	if err == nil {
		manuf, _ := core.ManufQuery(arpMac.String())
		res.ErrorMsg = "存活主机"
		res.Mac = arpMac.String()
		res.Manuf = manuf
		return syncDeployTaskResult(taskID, res)
	}

	linkName := fmt.Sprintf("hy_%s", random.RandStrV2(6))
	mac, err := ip_service.SetIp(ip_service.SetIpRequest{
		Ip:       info.Ip,
		Mask:     info.Mask,
		LinkName: linkName,
		Network:  req.Network,
	})
	if err != nil {
		res.ErrorMsg = err.Error()
		return syncDeployTaskResult(taskID, res)
	}

	res.Mac = mac
	res.LinkName = linkName

	// 本地资源创建成功后先写SQLite，便于重启恢复时做差集判断
	global.DB.Where(models.IpModel{Network: req.Network, Ip: info.Ip}).FirstOrCreate(&models.IpModel{
		Ip:       info.Ip,
		Mask:     info.Mask,
		LinkName: linkName,
		Network:  req.Network,
		Mac:      mac,
	})

	return syncDeployTaskResult(taskID, res)
}

// processUpdateTaskItem 执行单个更新项，并按检查点状态推进回调
func processUpdateTaskItem(taskID string, req models.BatchUpdateDeployRequest, ip string, portList []models.PortInfo, log *logrus.Entry) error {
	// 更新走全量覆盖，先关旧规则再按最新配置重建
	port_service.CloseIpTunnel(ip)

	res := UpdateDeployStatusRequest{
		NetID:    req.NetID,
		IP:       ip,
		LogID:    req.LogID,
		ErrorMsg: "",
	}

	for _, info := range portList {
		portStatus := PortInfo{
			Port: info.Port,
		}

		if err := ensurePortRule(info); err != nil {
			portStatus.ErrorMsg = err.Error()
			log.WithError(err).WithField("local_addr", info.LocalAddr()).Error("更新端口转发失败")
		}
		res.PortList = append(res.PortList, portStatus)
	}

	return syncUpdateTaskResult(taskID, res)
}

// processRemoveTaskItem 执行单个删除项，并按检查点状态推进回调
func processRemoveTaskItem(taskID string, req models.BatchRemoveDeployRequest, ip string, linkName string, log *logrus.Entry) error {
	// 删除前先关掉该IP下所有端口隧道，避免遗留入口流量
	port_service.CloseIpTunnel(ip)

	res := RemoveDeployStatusRequest{
		NetID:    req.NetID,
		IP:       ip,
		LogID:    req.LogID,
		ErrorMsg: "",
	}

	if err := ip_service.RemoveInterface(linkName); err != nil {
		res.ErrorMsg = err.Error()
		log.WithError(err).WithField("link_name", linkName).Error("删除虚拟网卡失败")
	}

	global.DB.Delete(&models.IpModel{}, "ip = ?", ip)
	return syncRemoveTaskResult(taskID, res)
}

// syncDeployTaskResult 按“本地完成 -> 发MQ -> 标记已回传”的顺序推进部署检查点
func syncDeployTaskResult(taskID string, res DeployStatusRequest) error {
	if err := task_checkpoint.UpdateTask(taskID, func(task *models.TaskModel) error {
		item := task.FindDeployItem(res.IP)
		if item == nil {
			return fmt.Errorf("deploy task item not found: %s", res.IP)
		}
		item.Stage = models.TaskItemStageLocalDone
		item.LinkName = res.LinkName
		item.Mac = res.Mac
		item.ErrorMsg = res.ErrorMsg
		item.Manuf = res.Manuf
		return nil
	}); err != nil {
		return err
	}

	if err := SendDeployStatusMsg(res); err != nil {
		return err
	}

	return task_checkpoint.UpdateTask(taskID, func(task *models.TaskModel) error {
		item := task.FindDeployItem(res.IP)
		if item == nil {
			return fmt.Errorf("deploy task item not found: %s", res.IP)
		}
		item.Stage = models.TaskItemStageReported
		return nil
	})
}

// syncUpdateTaskResult 按“本地完成 -> 发MQ -> 标记已回传”的顺序推进更新检查点
func syncUpdateTaskResult(taskID string, res UpdateDeployStatusRequest) error {
	if err := task_checkpoint.UpdateTask(taskID, func(task *models.TaskModel) error {
		item := task.FindUpdateItem(res.IP)
		if item == nil {
			return fmt.Errorf("update task item not found: %s", res.IP)
		}
		item.Stage = models.TaskItemStageLocalDone
		item.ErrorMsg = res.ErrorMsg
		item.PortList = buildUpdateTaskPortState(res.PortList)
		return nil
	}); err != nil {
		return err
	}

	if err := SendUpdateDeployStatusMsg(res); err != nil {
		return err
	}

	return task_checkpoint.UpdateTask(taskID, func(task *models.TaskModel) error {
		item := task.FindUpdateItem(res.IP)
		if item == nil {
			return fmt.Errorf("update task item not found: %s", res.IP)
		}
		item.Stage = models.TaskItemStageReported
		return nil
	})
}

// syncRemoveTaskResult 按“本地完成 -> 发MQ -> 标记已回传”的顺序推进删除检查点
func syncRemoveTaskResult(taskID string, res RemoveDeployStatusRequest) error {
	if err := task_checkpoint.UpdateTask(taskID, func(task *models.TaskModel) error {
		item := task.FindRemoveItem(res.IP)
		if item == nil {
			return fmt.Errorf("remove task item not found: %s", res.IP)
		}
		item.Stage = models.TaskItemStageLocalDone
		item.ErrorMsg = res.ErrorMsg
		return nil
	}); err != nil {
		return err
	}

	if err := SendRemoveDeployStatusMsg(res); err != nil {
		return err
	}

	return task_checkpoint.UpdateTask(taskID, func(task *models.TaskModel) error {
		item := task.FindRemoveItem(res.IP)
		if item == nil {
			return fmt.Errorf("remove task item not found: %s", res.IP)
		}
		item.Stage = models.TaskItemStageReported
		return nil
	})
}

// resendDeployTaskItem 对已经本地完成但尚未回传的部署项补发状态
func resendDeployTaskItem(taskID string, req models.BatchDeployRequest, item models.DeployTaskItemState) error {
	res := DeployStatusRequest{
		NetID:    req.NetID,
		IP:       item.IP,
		Mac:      item.Mac,
		LinkName: item.LinkName,
		LogID:    req.LogID,
		ErrorMsg: item.ErrorMsg,
		Manuf:    item.Manuf,
	}
	if err := SendDeployStatusMsg(res); err != nil {
		return err
	}
	return task_checkpoint.UpdateTask(taskID, func(task *models.TaskModel) error {
		target := task.FindDeployItem(item.IP)
		if target == nil {
			return fmt.Errorf("deploy task item not found: %s", item.IP)
		}
		target.Stage = models.TaskItemStageReported
		return nil
	})
}

// resendUpdateTaskItem 对已经本地完成但尚未回传的更新项补发状态
func resendUpdateTaskItem(taskID string, req models.BatchUpdateDeployRequest, item models.UpdateTaskItemState) error {
	res := UpdateDeployStatusRequest{
		NetID:    req.NetID,
		IP:       item.IP,
		LogID:    req.LogID,
		ErrorMsg: item.ErrorMsg,
		PortList: buildUpdateStatusPortList(item.PortList),
	}
	if err := SendUpdateDeployStatusMsg(res); err != nil {
		return err
	}
	return task_checkpoint.UpdateTask(taskID, func(task *models.TaskModel) error {
		target := task.FindUpdateItem(item.IP)
		if target == nil {
			return fmt.Errorf("update task item not found: %s", item.IP)
		}
		target.Stage = models.TaskItemStageReported
		return nil
	})
}

// resendRemoveTaskItem 对已经本地完成但尚未回传的删除项补发状态
func resendRemoveTaskItem(taskID string, req models.BatchRemoveDeployRequest, item models.RemoveTaskItemState) error {
	res := RemoveDeployStatusRequest{
		NetID:    req.NetID,
		IP:       item.IP,
		LogID:    req.LogID,
		ErrorMsg: item.ErrorMsg,
	}
	if err := SendRemoveDeployStatusMsg(res); err != nil {
		return err
	}
	return task_checkpoint.UpdateTask(taskID, func(task *models.TaskModel) error {
		target := task.FindRemoveItem(item.IP)
		if target == nil {
			return fmt.Errorf("remove task item not found: %s", item.IP)
		}
		target.Stage = models.TaskItemStageReported
		return nil
	})
}

// reconcileDeployTaskPorts 补齐批量部署任务缺失的端口规则
func reconcileDeployTaskPorts(req models.BatchDeployRequest, log *logrus.Entry) {
	for _, deployIP := range req.IPList {
		for _, port := range deployIP.PortList {
			if err := ensurePortRule(port); err != nil {
				log.WithError(err).WithField("local_addr", port.LocalAddr()).Error("补齐部署端口规则失败")
			}
		}
	}
}

// ensurePortRule 确保本地端口规则和Tunnel监听与目标配置一致
func ensurePortRule(port models.PortInfo) error {
	localAddr := port.LocalAddr()
	targetAddr := port.TargetAddr()

	var current models.PortModel
	err := global.DB.Take(&current, "local_addr = ?", localAddr).Error
	if err == nil && current.TargetAddr == targetAddr {
		// 启动时LoadTunnel已经会恢复同地址监听，配置完全一致时无需重复折腾
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	// 旧规则和目标不一致时先删旧记录，再按最新配置重建
	global.DB.Where("local_addr = ?", localAddr).Delete(&models.PortModel{})
	global.DB.Create(&models.PortModel{
		TargetAddr: targetAddr,
		LocalAddr:  localAddr,
	})

	return port_service.Tunnel(localAddr, targetAddr)
}

// updateTaskItemMatchesDesired 判断某个IP当前本地端口规则是否已经和目标规则一致
func updateTaskItemMatchesDesired(ip string, desired []models.PortInfo) bool {
	var portList []models.PortModel
	global.DB.Find(&portList, "local_addr LIKE ?", fmt.Sprintf("%s:%%", ip))
	if len(portList) != len(desired) {
		return false
	}

	currentMap := make(map[string]string, len(portList))
	for _, port := range portList {
		currentMap[port.LocalAddr] = port.TargetAddr
	}

	for _, port := range desired {
		if currentMap[port.LocalAddr()] != port.TargetAddr() {
			return false
		}
	}
	return true
}

// removeTaskItemCompleted 判断某个IP的本地删除是否已经完成
func removeTaskItemCompleted(item models.RemoveTaskItemState) bool {
	var ipModel models.IpModel
	global.DB.Find(&ipModel, "ip = ?", item.IP)
	if ipModel.ID != 0 {
		return false
	}

	var portCount int64
	global.DB.Model(&models.PortModel{}).Where("local_addr LIKE ?", fmt.Sprintf("%s:%%", item.IP)).Count(&portCount)
	if portCount > 0 {
		return false
	}

	if item.LinkName != "" {
		if networkMap, err := info2.GetNetworkInterfaces(); err == nil {
			if ips, ok := networkMap[item.LinkName]; ok && len(ips) > 0 {
				return false
			}
		}
	}

	// 再用本地真实IP兜底一次，避免SQLite缺记录但接口残留时误判完成
	return !info2.FindLocalIp(item.IP)
}

// findDeployInfo 查找批量部署任务中指定IP的原始请求数据
func findDeployInfo(req models.BatchDeployRequest, ip string) (models.DeployIp, bool) {
	for _, info := range req.IPList {
		if info.Ip == ip {
			return info, true
		}
	}
	return models.DeployIp{}, false
}

// findHoneyInterface 查找本机已经存在的诱捕虚拟网卡
func findHoneyInterface(ip string) (linkName string, mac string, ok bool) {
	networkMap, err := info2.GetNetworkInterfaces()
	if err != nil {
		return "", "", false
	}

	for iface, ips := range networkMap {
		if !strings.HasPrefix(iface, "hy_") {
			continue
		}
		for _, currentIP := range ips {
			if currentIP != ip {
				continue
			}

			mac, _ = ip_service.GetMACAddress(iface)
			return iface, mac, true
		}
	}

	return "", "", false
}

// buildUpdatePortMap 按IP分组构建更新目标端口列表
func buildUpdatePortMap(req models.BatchUpdateDeployRequest) map[string][]models.PortInfo {
	portMap := make(map[string][]models.PortInfo, len(req.IpList))
	for _, port := range req.PortList {
		portMap[port.IP] = append(portMap[port.IP], port)
	}
	for _, ip := range req.IpList {
		if _, ok := portMap[ip]; !ok {
			portMap[ip] = []models.PortInfo{}
		}
	}
	return portMap
}

// buildUpdateTaskPortState 将更新状态消息里的端口结果转成持久化快照
func buildUpdateTaskPortState(portList []PortInfo) []models.UpdateTaskPortState {
	list := make([]models.UpdateTaskPortState, 0, len(portList))
	for _, port := range portList {
		list = append(list, models.UpdateTaskPortState{
			Port:     port.Port,
			ErrorMsg: port.ErrorMsg,
		})
	}
	return list
}

// buildUpdateStatusPortList 将持久化快照恢复成更新状态消息中的端口结果
func buildUpdateStatusPortList(portList []models.UpdateTaskPortState) []PortInfo {
	list := make([]PortInfo, 0, len(portList))
	for _, port := range portList {
		list = append(list, PortInfo{
			Port:     port.Port,
			ErrorMsg: port.ErrorMsg,
		})
	}
	return list
}

// buildUpdatePortStatus 为“规则已经一致，只补回调”的场景构造成功端口列表
func buildUpdatePortStatus(portList []models.PortInfo) []PortInfo {
	list := make([]PortInfo, 0, len(portList))
	for _, port := range portList {
		list = append(list, PortInfo{
			Port: port.Port,
		})
	}
	return list
}
