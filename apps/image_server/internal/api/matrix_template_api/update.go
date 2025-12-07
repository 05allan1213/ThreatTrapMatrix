package matrix_template_api

// File: image_server/api/matrix_template_api/update.go
// Description: 矩阵模板更新API接口

import (
	"fmt"
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// UpdateRequest 矩阵模板更新请求参数结构体
type UpdateRequest struct {
	ID               uint                      `json:"id" binding:"required"`                             // 矩阵模板ID（必填）
	Title            string                    `json:"title" binding:"required"`                          // 新模板名称（保证唯一性）
	HostTemplateList []models.HostTemplateInfo `json:"hostTemplateList" binding:"required,dive,required"` // 更新后的主机模板列表（至少一个）
}

// UpdateView 矩阵模板更新接口处理函数
func (MatrixTemplateApi) UpdateView(c *gin.Context) {
	log := middleware.GetLog(c)

	// 获取并绑定矩阵模板更新请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	log.WithFields(map[string]interface{}{
		"matrix_template_id": cr.ID,
		"request_data":       cr,
	}).Info("matrix template update request received") // 收到矩阵模板更新请求

	// 校验待更新的矩阵模板是否存在
	var model models.MatrixTemplateModel
	if err := global.DB.Take(&model, cr.ID).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"matrix_template_id": cr.ID,
			"error":              err,
		}).Warn("matrix template not found") // 矩阵模板不存在
		response.FailWithMsg("矩阵模板不存在", c)
		return
	}

	// 校验主机模板列表不能为空
	if len(cr.HostTemplateList) == 0 {
		log.WithFields(map[string]interface{}{
			"matrix_template_id": cr.ID,
		}).Warn("no host templates associated with matrix template") // 没有与矩阵模板关联的主机模板
		response.FailWithMsg("矩阵模板需要关联至少一个主机模板", c)
		return
	}

	// 校验新模板名称的唯一性（排除自身ID）
	var duplicateModel models.MatrixTemplateModel
	if err := global.DB.Take(&duplicateModel, "title = ? and id <> ?", cr.Title, cr.ID).Error; err == nil {
		log.WithFields(map[string]interface{}{
			"matrix_template_id": cr.ID,
			"title":              cr.Title,
			"conflicting_id":     duplicateModel.ID,
		}).Warn("duplicate matrix template title found") // 发现重复的矩阵模板标题
		response.FailWithMsg("修改的矩阵模板名称不能重复", c)
		return
	}

	// 收集关联的主机模板ID并校验有效性
	var hostTemplateIDList []uint
	for _, h := range cr.HostTemplateList {
		hostTemplateIDList = append(hostTemplateIDList, h.HostTemplateID)
	}

	// 查询关联的主机模板记录并构建映射
	var hostTemps []models.HostTemplateModel
	if err := global.DB.Find(&hostTemps, "id in ?", hostTemplateIDList).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"matrix_template_id": cr.ID,
			"host_template_ids":  hostTemplateIDList,
			"error":              err,
		}).Error("failed to query host templates") // 查询主机模板失败
		response.FailWithMsg("查询主机模板失败", c)
		return
	}

	hostTempMap := make(map[uint]models.HostTemplateModel)
	for _, m := range hostTemps {
		hostTempMap[m.ID] = m
	}

	// 校验所有关联的主机模板是否存在
	for _, h := range cr.HostTemplateList {
		if _, ok := hostTempMap[h.HostTemplateID]; !ok {
			log.WithFields(map[string]interface{}{
				"matrix_template_id": cr.ID,
				"host_template_id":   h.HostTemplateID,
			}).Warn("referenced host template does not exist") // 关联的主机模板不存在
			response.FailWithMsg(fmt.Sprintf("主机模板 %d 不存在", h.HostTemplateID), c)
			return
		}
	}

	// 组装更新数据并执行更新操作
	updateData := models.MatrixTemplateModel{
		Title:            cr.Title,
		HostTemplateList: cr.HostTemplateList,
	}
	if err := global.DB.Model(&model).Updates(updateData).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"matrix_template_id": cr.ID,
			"update_data":        updateData,
			"error":              err,
		}).Error("failed to update matrix template") // 矩阵模板更新失败
		response.FailWithMsg("矩阵模板修改失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"matrix_template_id":  cr.ID,
		"new_title":           cr.Title,
		"host_template_count": len(cr.HostTemplateList),
	}).Info("matrix template updated successfully") // 矩阵模板更新成功

	response.OkWithMsg("矩阵模板修改成功", c)
}
