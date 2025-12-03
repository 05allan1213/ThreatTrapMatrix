package white_ip_api

// File: alert_server/api/white_ip_api/enter.go
// Description: 白名单IP管理API接口

import (
	"alert_server/internal/global"
	"alert_server/internal/middleware"
	"alert_server/internal/models"
	"alert_server/internal/service/common_service"
	"alert_server/internal/utils/response"
	"fmt"

	"github.com/gin-gonic/gin"
)

// WhiteIPApi 白名单IP管理接口统一入口结构体
type WhiteIPApi struct {
}

// CreateRequest 创建白名单IP的请求参数结构体
type CreateRequest struct {
	IP     string `json:"ip" binding:"required,ip"` // 白名单IP地址
	Notice string `json:"notice"`                   // 备注信息
}

// CreateView 创建白名单IP接口
func (WhiteIPApi) CreateView(c *gin.Context) {
	// 绑定并校验请求参数
	cr := middleware.GetBind[CreateRequest](c)

	// 校验IP是否已存在于白名单（避免重复添加）
	var model models.WhiteIPModel
	err := global.DB.Take(&model, "ip = ?", cr.IP).Error
	if err == nil {
		response.FailWithMsg("白名单ip不能重复", c)
		return
	}

	// 保存白名单IP数据到数据库
	err = global.DB.Create(&models.WhiteIPModel{
		IP:     cr.IP,
		Notice: cr.Notice,
	}).Error
	if err != nil {
		response.FailWithMsg("白名单ip保存失败", c)
		return
	}

	response.OkWithMsg("白名单ip保存成功", c)
}

// ListView 白名单IP列表查询接口
func (WhiteIPApi) ListView(c *gin.Context) {
	// 绑定分页查询参数
	cr := middleware.GetBind[models.PageInfo](c)

	// 调用通用查询服务，查询白名单IP列表
	list, count, _ := common_service.QueryList(models.WhiteIPModel{}, common_service.QueryListRequest{
		Likes:    []string{"ip", "notice"}, // 支持按IP和备注模糊搜索
		Sort:     "created_at desc",        // 按创建时间倒序排序
		PageInfo: cr,                       // 分页参数
	})

	// 返回分页列表结果
	response.OkWithList(list, count, c)
}

// UpdateRequest 更新白名单IP的请求参数结构体
type UpdateRequest struct {
	ID     uint   `json:"id" binding:"required"`    // 白名单ID
	IP     string `json:"ip" binding:"required,ip"` // 新的白名单IP地址
	Notice string `json:"notice"`                   // 新的备注信息
}

// UpdateView 更新白名单IP接口
func (WhiteIPApi) UpdateView(c *gin.Context) {
	// 绑定并校验请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	// 校验待更新的白名单记录是否存在
	var model models.WhiteIPModel
	err := global.DB.Take(&model, cr.ID).Error
	if err != nil {
		response.FailWithMsg("白名单ip不存在", c)
		return
	}

	// 校验新IP是否已被其他白名单记录占用（排除当前更新的记录）
	var newModel models.WhiteIPModel
	err = global.DB.Take(&newModel, "id <> ? and ip = ?", cr.ID, cr.IP).Error
	if err == nil {
		response.FailWithMsg("修改的ip不能重复", c)
		return
	}

	// 更新白名单IP及备注信息
	err = global.DB.Model(&model).Updates(map[string]any{
		"ip":     cr.IP,
		"notice": cr.Notice,
	}).Error
	if err != nil {
		response.FailWithMsg("白名单ip更新失败", c)
		return
	}

	response.OkWithMsg("白名单ip更新成功", c)
}

// RemoveView 批量删除白名单IP接口
func (WhiteIPApi) RemoveView(c *gin.Context) {
	// 绑定批量删除的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)
	// 获取请求日志实例（用于记录操作日志）
	log := middleware.GetLog(c)

	// 调用通用删除服务，批量删除白名单IP
	successCount, err := common_service.Remove(models.WhiteIPModel{}, common_service.RemoveRequest{
		IDList: cr.IdList,  // 待删除的白名单ID列表
		Log:    log,        // 日志实例
		Msg:    "白名单ip", // 业务模块名称（用于日志记录）
	})
	if err != nil {
		msg := fmt.Sprintf("删除白名单ip失败 %s", err)
		response.FailWithMsg(msg, c)
		return
	}

	// 返回删除结果（总数量、成功数量）
	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", len(cr.IdList), successCount)
	response.OkWithMsg(msg, c)
}
