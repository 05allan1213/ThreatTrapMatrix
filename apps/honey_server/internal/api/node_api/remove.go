package node_api

// File: honey_server/api/node_api/remove.go
// Description: 节点删除API接口实现

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
)

// RemoveView 节点删除接口处理函数
func (NodeApi) RemoveView(c *gin.Context) {
	// 获取请求关联的日志实例（含traceID），用于全流程日志追踪
	log := middleware.GetLog(c)
	// 绑定并解析前端提交的节点ID请求参数
	cr := middleware.GetBind[models.IDRequest](c)

	// 记录节点删除请求接收日志，包含目标节点ID
	log.WithFields(map[string]interface{}{
		"node_id": cr.Id,
	}).Info("node deletion request received")

	// 查询目标节点的数据库记录，校验节点是否存在
	var model models.NodeModel
	if err := global.DB.Take(&model, cr.Id).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"node_id": cr.Id,
			"error":   err,
		}).Warn("node not found")
		response.FailWithMsg("节点不存在", c)
		return
	}

	// 从数据库中删除节点记录，失败则返回错误
	if err := global.DB.Delete(&model).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"node_id": cr.Id,
			"error":   err,
		}).Error("database deletion failed")
		response.FailWithMsg("节点删除失败", c)
		return
	}

	// 启动独立协程异步下发节点移除RPC命令，避免阻塞HTTP响应
	go func(uid string, logID string) {
		// 获取节点对应的RPC命令通道（包含请求/响应通道），校验节点是否在线
		cmd, ok := grpc_service.GetNodeCommand(model.Uid)
		if !ok {
			log.WithFields(map[string]interface{}{
				"node_uid": model.Uid,
			}).Warn("节点离线")
			return
		}

		// 构造节点移除RPC命令请求
		req := &node_rpc.CmdRequest{
			CmdType:             node_rpc.CmdType_cmdNodeRemoveType,                  // 命令类型：节点移除
			TaskID:              fmt.Sprintf("nodeRemove-%d", time.Now().UnixNano()), // 任务ID：保证唯一性，用于关联请求/响应
			LogID:               logID,                                               // 日志ID：用于关联请求/响应
			NodeRemoveInMessage: &node_rpc.NodeRemoveInMessage{},                     // 节点移除入参
		}

		// 创建带30秒超时的上下文，防止RPC请求/响应阻塞协程
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel() // 函数退出时释放上下文资源，避免内存泄漏

		// 发送RPC请求到节点的请求通道，带超时控制
		select {
		case cmd.ReqChan <- req:
			// 请求发送成功，记录日志
			log.WithFields(map[string]interface{}{
				"node_uid": uid,
				"task_id":  req.TaskID,
			}).Info("删除节点消息发送成功")
		case <-ctx.Done():
			// 请求发送超时，记录错误日志并退出协程
			log.WithFields(map[string]interface{}{
				"node_uid": uid,
				"error":    ctx.Err(),
			}).Error("删除节点消息发送超时")
			return
		}

		// 监听节点的RPC响应通道，接收移除命令执行结果，带超时控制
		select {
		case res := <-cmd.ResChan:
			// 接收响应成功，记录日志
			log.WithFields(map[string]interface{}{
				"node_uid": uid,
				"task_id":  req.TaskID,
				"response": res.NodeRemoveOutMessage,
			}).Info("删除节点消息接收成功")

		case <-ctx.Done():
			// 接收响应超时，记录错误日志
			log.WithFields(map[string]interface{}{
				"node_uid": uid,
				"error":    ctx.Err(),
			}).Error("删除节点消息接收超时")
			response.FailWithMsg("获取响应超时", c)
			return
		}
	}(model.Uid, log.Data["logID"].(string))

	// 记录节点删除成功日志（数据库层面）
	log.WithFields(map[string]interface{}{
		"node_id": cr.Id,
	}).Info("node deleted successfully")

	// 返回HTTP成功响应
	response.OkWithMsg("节点删除成功", c)
}
