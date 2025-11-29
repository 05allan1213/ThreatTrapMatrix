package matrix_template_api

// File: image_server/api/matrix_template_api/update.go
// Description: 矩阵模板更新API接口

import (
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"
	"fmt"

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
	// 获取并绑定矩阵模板更新请求参数
	cr := middleware.GetBind[UpdateRequest](c)

	// 校验待更新的矩阵模板是否存在
	var model models.MatrixTemplateModel
	err := global.DB.Take(&model, cr.ID).Error
	if err != nil {
		response.FailWithMsg("矩阵模板不存在", c)
		return
	}

	// 校验主机模板列表不能为空
	if len(cr.HostTemplateList) == 0 {
		response.FailWithMsg("矩阵模板需要关联至少一个主机模板", c)
		return
	}

	// 校验新模板名称的唯一性（排除自身ID）
	var newModel models.MatrixTemplateModel
	err = global.DB.Take(&newModel, "title = ? and id <> ?", cr.Title, cr.ID).Error
	if err == nil {
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
	global.DB.Find(&hostTemps, "id in ?", hostTemplateIDList)
	var hostTempMap = map[uint]models.HostTemplateModel{}
	for _, m := range hostTemps {
		hostTempMap[m.ID] = m
	}

	// 校验所有关联的主机模板是否存在
	for _, h := range cr.HostTemplateList {
		_, ok := hostTempMap[h.HostTemplateID]
		if !ok {
			msg := fmt.Sprintf("主机模板 %d 不存在", h.HostTemplateID)
			response.FailWithMsg(msg, c)
			return
		}
	}

	// 组装更新数据并执行更新操作
	newModel = models.MatrixTemplateModel{
		Title:            cr.Title,
		HostTemplateList: cr.HostTemplateList,
	}
	err = global.DB.Model(&model).Updates(newModel).Error
	if err != nil {
		response.FailWithMsg("矩阵模板修改失败", c)
		return
	}

	response.OkWithMsg("矩阵模板修改成功", c)
}
