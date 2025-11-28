package matrix_template_api

// File: image_server/api/matrix_template_api/create.go
// Description: 矩阵模板创建API接口

import (
	"ThreatTrapMatrix/apps/image_server/internal/global"
	"ThreatTrapMatrix/apps/image_server/internal/middleware"
	"ThreatTrapMatrix/apps/image_server/internal/models"
	"ThreatTrapMatrix/apps/image_server/internal/utils/response"
	"fmt"

	"github.com/gin-gonic/gin"
)

// CreateRequest 矩阵模板创建请求参数结构体
type CreateRequest struct {
	Title            string                    `json:"title" binding:"required"`                          // 矩阵模板名称（必需）
	HostTemplateList []models.HostTemplateInfo `json:"hostTemplateList" binding:"required,dive,required"` // 关联的主机模板列表（至少一个）
}

// CreateView 矩阵模板创建接口处理函数
func (MatrixTemplateApi) CreateView(c *gin.Context) {
	// 获取并绑定矩阵模板创建请求参数
	cr := middleware.GetBind[CreateRequest](c)

	// 校验主机模板列表不能为空
	if len(cr.HostTemplateList) == 0 {
		response.FailWithMsg("矩阵模板需要关联至少一个主机模板", c)
		return
	}

	// 校验矩阵模板名称唯一性
	var model models.MatrixTemplateModel
	err := global.DB.Take(&model, "title = ? ", cr.Title).Error
	if err == nil {
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

	// 组装矩阵模板数据并入库
	model = models.MatrixTemplateModel{
		Title:            cr.Title,
		HostTemplateList: cr.HostTemplateList,
	}
	err = global.DB.Create(&model).Error
	if err != nil {
		response.FailWithMsg("矩阵模板创建失败", c)
		return
	}

	// 返回创建成功的模板ID
	response.OkWithData(model.ID, c)
}
