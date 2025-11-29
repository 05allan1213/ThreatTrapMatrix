package node_network_api

// File: honey_server/api/node_network_api/flush.go
// Description: 节点网卡刷新API接口

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// FlushView 处理节点网络视图刷新请求，向指定节点发送网卡刷新命令并返回结果
func (NodeNetworkApi) FlushView(c *gin.Context) {
	// 从请求中绑定并获取节点ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	var model models.NodeModel
	// 查询指定ID的节点信息
	if err := global.DB.Take(&model, cr.Id).Error; err != nil {
		response.FailWithMsg("节点不存在", c)
		return
	}

	// 检查节点状态是否为运行中
	if model.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 通过节点唯一标识获取RPC命令通道
	cmd, ok := grpc_service.GetNodeCommand(model.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 构建网卡刷新RPC请求，包含常见虚拟网卡过滤规则
	req := &node_rpc.CmdRequest{
		CmdType: node_rpc.CmdType_cmdNetworkFlushType,           // 命令类型：网卡刷新
		TaskID:  fmt.Sprintf("flush-%d", time.Now().UnixNano()), // 生成唯一任务ID
		NetworkFlushInMessage: &node_rpc.NetworkFlushInMessage{
			// 刷新过滤虚拟网卡前缀，避免获取诱捕网卡信息
			FilterNetworkName: []string{"hy-"},
		},
	}

	// 创建带30秒超时的上下文，控制RPC请求生命周期
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 发送网卡刷新命令到节点
	select {
	case cmd.ReqChan <- req:
		logrus.Debugf("已向节点 %s 发送网卡刷新请求", model.Uid)
	case <-ctx.Done():
		response.FailWithMsg("发送命令超时", c)
		return
	}

	// 定义变量接收节点返回的网卡信息列表
	var networkInfoList []*node_rpc.NetworkInfoMessage
	// 等待节点响应结果
	select {
	case res := <-cmd.ResChan:
		logrus.Debugf("已接收节点 %s 的网卡刷新响应", model.Uid)
		networkInfoList = res.NetworkFlushOutMessage.NetworkList // 提取网卡信息
	case <-ctx.Done():
		response.FailWithMsg("获取响应超时", c)
		return
	}

	// 查询数据库中当前节点的网卡记录
	var networkList []models.NodeNetworkModel
	global.DB.Find(&networkList, "node_id = ?", model.ID)

	// 构建现有网卡名称到索引的映射，用于快速对比
	networkMap := make(map[string]int)
	for i, network := range networkList {
		networkMap[network.Network] = i
	}

	// 构建新网卡名称到索引的映射
	newNetworkMap := make(map[string]int)
	for i, network := range networkInfoList {
		logrus.Infof("节点网卡信息：%s %s", network.Network, network.Net)
		newNetworkMap[network.Network] = i
	}

	// 计算新增网卡（新列表有，数据库无）
	var newNetworks []*node_rpc.NetworkInfoMessage
	for networkName := range newNetworkMap {
		if _, exists := networkMap[networkName]; !exists {
			newNetworks = append(newNetworks, networkInfoList[newNetworkMap[networkName]])
		}
	}

	// 计算待删除网卡（数据库有，新列表无）
	var deletedNetworks []models.NodeNetworkModel
	for networkName := range networkMap {
		if _, exists := newNetworkMap[networkName]; !exists {
			deletedNetworks = append(deletedNetworks, networkList[networkMap[networkName]])
		}
	}

	// 计算待更新网卡（存在但IP/Mask变化）
	var updatedNetworks []models.NodeNetworkModel
	for networkName := range networkMap {
		if newIndex, exists := newNetworkMap[networkName]; exists {
			dbNetwork := networkList[networkMap[networkName]]
			newNetwork := networkInfoList[newIndex]

			// 检查IP或子网掩码是否变更
			if dbNetwork.IP != newNetwork.Ip || dbNetwork.Mask != int8(newNetwork.Mask) {
				dbNetwork.IP = newNetwork.Ip
				dbNetwork.Mask = int8(newNetwork.Mask)
				updatedNetworks = append(updatedNetworks, dbNetwork)
			}
		}
	}

	// 执行数据库新增操作
	for _, network := range newNetworks {
		newRecord := models.NodeNetworkModel{
			NodeID:  model.ID,
			Network: network.Network,
			IP:      network.Ip,
			Mask:    int8(network.Mask),
			Status:  2, // 初始状态设为未启用
		}
		if err := global.DB.Create(&newRecord).Error; err != nil {
			logrus.Errorf("创建网卡记录失败: %v", err)
		}
	}

	// 执行数据库删除操作
	for _, network := range deletedNetworks {
		if err := global.DB.Delete(&network).Error; err != nil {
			logrus.Errorf("删除网卡记录失败: %v", err)
		}
	}

	// 执行数据库更新操作
	for _, network := range updatedNetworks {
		if err := global.DB.Save(&network).Error; err != nil {
			logrus.Errorf("更新网卡记录失败: %v", err)
		}
	}

	// 记录网卡信息同步统计
	logrus.Infof("节点网卡同步完成：新增 %d 张，删除 %d 张，更新 %d 张", len(newNetworks), len(deletedNetworks), len(updatedNetworks))

	// 返回成功响应
	response.OkWithMsg("网卡信息更新成功", c)
}
