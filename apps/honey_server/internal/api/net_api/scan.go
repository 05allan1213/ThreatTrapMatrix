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
	// 获取请求绑定的ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	var model models.NetModel
	// 查询网络模型并预加载关联的节点信息
	if err := global.DB.Preload("NodeModel").Take(&model, cr.Id).Error; err != nil {
		response.FailWithMsg("网络不存在", c)
		return
	}

	// 检查节点是否处于运行状态
	if model.NodeModel.Status != 1 {
		response.FailWithMsg("节点未运行", c)
		return
	}

	// 使用封装的获取节点函数，获取指定节点的命令通道
	cmd, ok := grpc_service.GetNodeCommand(model.NodeModel.Uid)
	if !ok {
		response.FailWithMsg("节点离线中", c)
		return
	}

	// 构建网络扫描请求参数
	req := &node_rpc.CmdRequest{
		CmdType: node_rpc.CmdType_cmdNetScanType,
		TaskID:  fmt.Sprintf("netScan-%d", time.Now().UnixNano()), // 生成唯一任务ID
		NetScanInMessage: &node_rpc.NetScanInMessage{
			Network:      model.Network,            // 扫描使用的网络接口
			IpRange:      model.CanUseHoneyIPRange, // 待扫描的IP范围
			FilterIPList: []string{},               // 过滤IP列表（暂为空）
			NetID:        uint32(model.ID),         // 关联的网络ID
		},
	}

	// 创建带超时的上下文，设置30秒超时时间
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 发送扫描请求到节点命令通道
	select {
	case cmd.ReqChan <- req:
		logrus.Debugf("Sent flush request to node %s", model.NodeModel.Uid)
	case <-ctx.Done():
		response.FailWithMsg("发送命令超时", c)
		return
	}

label:
	// 循环接收节点的扫描响应结果
	for {
		select {
		case res := <-cmd.ResChan:
			logrus.Debugf("Received flush response from node %s", model.NodeModel.Uid)
			message := res.NetScanOutMessage
			fmt.Printf("网络扫描 数据 %v\n", message)

			// 检查扫描过程中是否出现错误
			if message.ErrMsg != "" {
				response.FailWithMsg("扫描错误"+message.ErrMsg, c)
				return
			}

			// 扫描完成时跳出循环
			if message.End {
				break label
			}
		case <-ctx.Done():
			response.FailWithMsg("获取响应超时", c)
			return
		}
	}

	// 扫描成功完成，返回成功响应
	response.OkWithMsg("扫描成功", c)
}
