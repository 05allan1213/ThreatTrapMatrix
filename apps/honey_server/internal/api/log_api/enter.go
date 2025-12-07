package log_api

// File: honey_server/api/log_api/enter.go
// Description: 日志模块API接口定义，提供日志列表查询、日志删除等HTTP接口处理逻辑

import (
	"fmt"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// LogApi 日志模块API处理器结构体
type LogApi struct{}

// LogListRequest 日志列表查询请求参数结构体
type LogListRequest struct {
	models.PageInfo        // 分页信息（包含Page、PageSize字段）
	Type            int8   `form:"type"` // 日志类型：1-登录日志
	IP              string `form:"ip"` // 日志关联IP地址
	Addr            string `form:"addr"` // 日志关联地址信息
}

// LogListView 日志列表查询接口处理方法
func (LogApi) LogListView(c *gin.Context) {
	// 获取并绑定请求参数
	cr := middleware.GetBind[LogListRequest](c)
	// 调用公共服务查询日志列表，支持按用户名模糊搜索，按创建时间降序排序
	list, count, _ := common_service.QueryList(models.LogModel{
		Type: cr.Type,
		IP:   cr.IP,
		Addr: cr.Addr,
	}, common_service.QueryListRequest{
		Likes:    []string{"username"}, // 用户名字段支持模糊查询
		PageInfo: cr.PageInfo,          // 分页参数
		Sort:     "created_at desc",    // 排序规则
	})
	// 返回带分页的列表数据
	response.OkWithList(list, count, c)
}

// RemoveView 日志删除接口处理方法
func (LogApi) RemoveView(c *gin.Context) {
	// 获取并绑定ID列表请求参数
	cr := middleware.GetBind[models.IDListRequest](c)
	// 获取上下文日志实例
	log := middleware.GetLog(c)
	// 调用公共服务执行日志删除操作（物理删除）
	log.WithFields(map[string]interface{}{
		"log_ids":     cr.IdList,
		"total_count": len(cr.IdList),
	}).Info("log deletion request received") // 收到日志删除请求

	successCount, err := common_service.Remove(
		models.LogModel{},
		common_service.RemoveRequest{
			IDList:   cr.IdList,
			Log:      log,
			Msg:      "日志",
			Unscoped: true,
		},
	)
	// 处理删除异常
	if err != nil {
		log.WithFields(map[string]interface{}{
			"log_ids": cr.IdList,
			"error":   err,
		}).Error("failed to delete logs") // 删除日志失败
		msg := fmt.Sprintf("删除日志失败 %s", err)
		response.FailWithMsg(msg, c)
		return
	}

	log.WithFields(map[string]interface{}{
		"log_ids":         cr.IdList,
		"total_requested": len(cr.IdList),
		"success_count":   successCount,
	}).Info("logs deletion completed successfully") // 日志删除成功
	// 返回删除成功结果
	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", len(cr.IdList), successCount)
	response.OkWithMsg(msg, c)
}
