package white_ip_api

// File: alert_server/api/white_ip_api/enter.go
// Description: 白名单IP管理API接口

import (
	"alert_server/internal/global"
	"alert_server/internal/middleware"
	"alert_server/internal/models"
	"alert_server/internal/service/common_service"
	"alert_server/internal/utils/response"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	log := middleware.GetLog(c)
	// 绑定并校验请求参数
	cr := middleware.GetBind[CreateRequest](c)

	// 校验IP是否已存在于白名单（避免重复添加）
	log.WithFields(map[string]interface{}{
		"ip":     cr.IP,
		"notice": cr.Notice,
	}).Info("white IP creation request received") // 收到白名单IP创建请求

	// 检查IP是否已存在
	var existingModel models.WhiteIPModel
	if err := global.DB.Take(&existingModel, "ip = ?", cr.IP).Error; err == nil {
		log.WithFields(map[string]interface{}{
			"ip":          cr.IP,
			"existing_id": existingModel.ID,
		}).Warn("white IP already exists") // 白名单IP已存在
		response.FailWithMsg("白名单ip不能重复", c)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 处理非"记录不存在"的数据库错误
		log.WithFields(map[string]interface{}{
			"ip":    cr.IP,
			"error": err,
		}).Error("failed to check existing white IP") // 未能检查到现有的白名单IP
		response.FailWithMsg("检查白名单IP是否存在失败", c)
		return
	}

	// 保存白名单IP数据到数据库
	newModel := models.WhiteIPModel{
		IP:     cr.IP,
		Notice: cr.Notice,
	}
	if err := global.DB.Create(&newModel).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"ip":    cr.IP,
			"error": err,
		}).Error("failed to create white IP") // 保存白名单IP数据到数据库失败
		response.FailWithMsg("白名单ip保存失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"white_ip_id": newModel.ID,
		"ip":          newModel.IP,
	}).Info("white IP created successfully") // 白名单IP保存成功
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
	log := middleware.GetLog(c)
	// 绑定并校验请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	log.WithFields(map[string]interface{}{
		"white_ip_id": cr.ID,
		"new_ip":      cr.IP,
		"new_notice":  cr.Notice,
	}).Info("white IP update request received") // 收到白名单IP更新请求

	// 校验待更新的白名单记录是否存在
	var model models.WhiteIPModel
	if err := global.DB.Take(&model, cr.ID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"white_ip_id": cr.ID,
			"error":       err,
		}).Warn("white IP not found") // 白名单IP不存在
		response.FailWithMsg("白名单ip不存在", c)
		return
	}

	// 校验新IP是否已被其他白名单记录占用（排除当前更新的记录）
	if cr.IP != model.IP { // 仅当IP发生变更时才检查重复
		var duplicateModel models.WhiteIPModel
		if err := global.DB.Take(&duplicateModel, "id <> ? and ip = ?", cr.ID, cr.IP).Error; err == nil {
			log.WithFields(map[string]interface{}{
				"white_ip_id": cr.ID,
				"conflict_ip": cr.IP,
				"conflict_id": duplicateModel.ID,
			}).Warn("duplicate white IP found") // 存在重复IP
			response.FailWithMsg("修改的ip不能重复", c)
			return
		}
	}

	// 更新白名单IP及备注信息
	updateData := map[string]any{
		"ip":     cr.IP,
		"notice": cr.Notice,
	}

	if err := global.DB.Model(&model).Updates(updateData).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"white_ip_id": cr.ID,
			"update_data": updateData,
			"error":       err,
		}).Error("failed to update white IP") // 白名单IP更新失败
		response.FailWithMsg("白名单ip更新失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"white_ip_id": cr.ID,
	}).Info("white IP updated successfully") // 白名单IP更新成功
	response.OkWithMsg("白名单ip更新成功", c)
}

// RemoveView 批量删除白名单IP接口
func (WhiteIPApi) RemoveView(c *gin.Context) {
	// 绑定批量删除的ID列表参数
	cr := middleware.GetBind[models.IDListRequest](c)
	// 获取请求日志实例（用于记录操作日志）
	log := middleware.GetLog(c)

	// 调用通用删除服务，批量删除白名单IP
	log.WithFields(map[string]interface{}{
		"white_ip_ids": cr.IdList,
		"total_count":  len(cr.IdList),
	}).Info("white IP removal request received") // 收到白名单IP批量删除请求

	successCount, err := common_service.Remove(
		models.WhiteIPModel{},
		common_service.RemoveRequest{
			IDList: cr.IdList,
			Log:    log,
			Msg:    "白名单ip",
		},
	)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"white_ip_ids": cr.IdList,
			"error":        err,
		}).Error("failed to delete white IPs") // 删除白名单IP失败
		msg := fmt.Sprintf("删除白名单ip失败 %s", err)
		response.FailWithMsg(msg, c)
		return
	}

	log.WithFields(map[string]interface{}{
		"white_ip_ids":    cr.IdList,
		"total_requested": len(cr.IdList),
		"success_count":   successCount,
	}).Info("white IPs deletion completed successfully") // 白名单IP删除成功
	// 返回删除结果（总数量、成功数量）
	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", len(cr.IdList), successCount)
	response.OkWithMsg(msg, c)
}

// RemoveByIpRequest 根据IP批量删除白名单IP的请求参数结构体
type RemoveByIpRequest struct {
	Ip string `json:"ip" binding:"required"`
}

// RemoveByIpView 根据IP批量删除白名单IP接口
func (WhiteIPApi) RemoveByIpView(c *gin.Context) {
	cr := middleware.GetBind[RemoveByIpRequest](c)
	log := middleware.GetLog(c)

	log.WithFields(map[string]interface{}{
		"ip": cr.Ip,
	}).Info("white IP removal request received")

	var model models.WhiteIPModel
	err := global.DB.Take(&model, "ip = ?", cr.Ip).Error
	if err != nil {
		response.FailWithMsg("白名单ip不存在", c)
		return
	}

	result := global.DB.Delete(&model)
	if result.Error != nil {
		log.WithFields(map[string]interface{}{
			"ip":    cr.Ip,
			"error": result.Error,
		}).Error("white IP remove error")
		response.FailWithMsg("删除白名单失败", c)
		return
	}

	msg := fmt.Sprintf("删除成功 共%d个，成功%d个", 1, result.RowsAffected)
	response.OkWithMsg(msg, c)
}
