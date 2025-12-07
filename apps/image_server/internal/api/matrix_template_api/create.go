package matrix_template_api

// File: image_server/api/matrix_template_api/create.go
// Description: 矩阵模板创建API接口

import (
	"fmt"
	"image_server/internal/global"
	"image_server/internal/middleware"
	"image_server/internal/models"
	"image_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// CreateRequest 矩阵模板创建请求参数结构体
type CreateRequest struct {
	Title            string                    `json:"title" binding:"required"`                          // 矩阵模板名称（必需）
	HostTemplateList []models.HostTemplateInfo `json:"hostTemplateList" binding:"required,dive,required"` // 关联的主机模板列表（至少一个）
}

// CreateView 矩阵模板创建接口处理函数
func (MatrixTemplateApi) CreateView(c *gin.Context) {
	log := middleware.GetLog(c)
	// 获取并绑定矩阵模板创建请求参数
	cr := middleware.GetBind[CreateRequest](c)

	log.WithFields(map[string]interface{}{
		"request_data": cr,
	}).Info("matrix template creation request received") // 收到矩阵模板创建请求

	// 校验主机模板列表不能为空
	if len(cr.HostTemplateList) == 0 {
		log.Warn("matrix template creation failed: no host templates associated") //
		response.FailWithMsg("矩阵模板需要关联至少一个主机模板", c)
		return
	}

	// 校验矩阵模板名称唯一性
	var existingModel models.MatrixTemplateModel
	if err := global.DB.Take(&existingModel, "title = ?", cr.Title).Error; err == nil {
		log.WithFields(map[string]interface{}{
			"title": cr.Title,
		}).Warn("duplicate matrix template title found") // 找到重复的矩阵模板名称
		response.FailWithMsg("矩阵模板名称不能重复", c)
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
			"host_template_ids": hostTemplateIDList,
			"error":             err,
		}).Error("failed to query host templates") // 主机模板查询失败
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
				"invalid_host_template_id": h.HostTemplateID,
			}).Warn("referenced host template does not exist") // 引用的主机模板不存在
			response.FailWithMsg(fmt.Sprintf("主机模板 %d 不存在", h.HostTemplateID), c)
			return
		}
	}

	// 组装矩阵模板数据并入库
	model := models.MatrixTemplateModel{
		Title:            cr.Title,
		HostTemplateList: cr.HostTemplateList,
	}
	if err := global.DB.Create(&model).Error; err != nil {
		log.WithFields(map[string]interface{}{
			"model_data": model,
			"error":      err,
		}).Error("failed to create matrix template in database") // 矩阵模板入库失败
		response.FailWithMsg("矩阵模板创建失败", c)
		return
	}

	log.WithFields(map[string]interface{}{
		"matrix_template_id":  model.ID,
		"title":               model.Title,
		"host_template_count": len(model.HostTemplateList),
	}).Info("matrix template created successfully") // 矩阵模板创建成功

	// 返回创建成功的模板ID
	response.OkWithData(model.ID, c)
}
