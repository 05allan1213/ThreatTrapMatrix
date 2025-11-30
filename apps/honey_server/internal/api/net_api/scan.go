package net_api

// File: honey_server/api/net_api/scan.go
// Description: 网络扫描API接口，通过RPC与节点交互执行网络扫描任务并处理响应

import (
	"context"
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/rpc/node_rpc"
	"honey_server/internal/service/grpc_service"
	"honey_server/internal/utils/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ScanView 处理网络扫描请求，通过RPC指令触发节点执行指定网络的扫描任务
func (NetApi) ScanView(c *gin.Context) {
	// 绑定并解析请求中的网络ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	var model models.NetModel
	// 查询指定ID的网络信息，并预加载关联的节点模型
	if err := global.DB.Preload("NodeModel").Take(&model, cr.Id).Error; err != nil {
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 校验关联节点是否处于运行状态
	if model.NodeModel.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 通过节点唯一标识获取RPC命令通道
	cmd, ok := grpc_service.GetNodeCommand(model.NodeModel.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 构建网络扫描RPC请求
	req := &node_rpc.CmdRequest{
		CmdType: node_rpc.CmdType_cmdNetScanType,                  // 命令类型：网络扫描
		TaskID:  fmt.Sprintf("netScan-%d", time.Now().UnixNano()), // 生成唯一任务标识
		NetScanInMessage: &node_rpc.NetScanInMessage{
			Network:      model.Network,            // 目标网络段
			IpRange:      model.CanUseHoneyIPRange, // 扫描IP范围
			FilterIPList: []string{},               // IP过滤列表
			NetID:        uint32(model.ID),         // 网络ID关联
		},
	}

	// 创建带30秒超时的上下文，控制RPC交互生命周期
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 发送扫描命令到节点
	select {
	case cmd.ReqChan <- req:
		logrus.Debugf("已向节点 %s 发送网络扫描请求", model.NodeModel.Uid)
	case <-ctx.Done():
		response.FailWithMsg("发送命令超时", c)
		return
	}

	// 循环接收节点扫描响应，直到收到结束标识
label:
	for {
		select {
		case res := <-cmd.ResChan:
			logrus.Debugf("已接收节点 %s 的扫描响应", model.NodeModel.Uid)
			message := res.NetScanOutMessage // 提取扫描响应数据
			fmt.Printf("网络扫描数据: %v\n", message)
			// 检测扫描任务结束标识，退出循环
			if message.End {
				break label
			}
		case <-ctx.Done():
			response.FailWithMsg("获取响应超时", c)
			return
		}
	}

	response.OkWithMsg("扫描成功", c)
}
