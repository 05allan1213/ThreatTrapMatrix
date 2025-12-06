package api

// File: matrix_server/api/select_matrix_template.go
// Description: 选择矩阵模板API接口

import (
	"math/rand"
	"matrix_server/internal/global"
	"matrix_server/internal/middleware"
	"matrix_server/internal/models"
	"matrix_server/internal/utils/response"
	"time"

	"github.com/gin-gonic/gin"
)

// SelectMatrixTemplateRequest 矩阵模板IP分配的请求参数结构体
type SelectMatrixTemplateRequest struct {
	MatrixTemplateID uint     `json:"matrixTemplateID" binding:"required"` // 矩阵模板ID
	IpList           []string `json:"ipList" binding:"required,dive,ip"`   // 待分配的IP列表
}

// SelectMatrixTemplateView 矩阵模板IP分配接口处理函数
func (Api) SelectMatrixTemplateView(c *gin.Context) {
	// 绑定并解析请求参数到SelectMatrixTemplateRequest结构体
	cr := middleware.GetBind[SelectMatrixTemplateRequest](c)
	// 查询矩阵模板信息，校验模板是否存在
	var model models.MatrixTemplateModel
	err := global.DB.Take(&model, cr.MatrixTemplateID).Error
	if err != nil {
		response.FailWithMsg("矩阵模板不存在", c)
		return
	}

	// 初始化随机数种子，保证模板打乱顺序的随机性
	rand.Seed(time.Now().UnixNano())

	// 计算矩阵模板下所有主机模板的总权重（为后续按权重分配IP做基础）
	totalWeight := 0
	for _, template := range model.HostTemplateList {
		totalWeight += template.Weight
	}

	// 打乱主机模板的顺序（避免固定分配顺序，增加分配随机性）
	shuffledTemplates := make(models.HostTemplateList, len(model.HostTemplateList))
	copy(shuffledTemplates, model.HostTemplateList)
	rand.Shuffle(len(shuffledTemplates), func(i, j int) {
		shuffledTemplates[i], shuffledTemplates[j] = shuffledTemplates[j], shuffledTemplates[i]
	})

	// 初始化映射：记录每个主机模板应分配的IP数量
	ipCountPerTemplate := make(map[uint]int)
	remainingIPs := len(cr.IpList)               // 待分配的剩余IP数量
	remainingTemplates := len(shuffledTemplates) // 剩余未分配的主机模板数量

	// 遍历打乱后的模板，计算每个模板应分配的IP数量（按权重比例）
	for i, template := range shuffledTemplates {
		// 按权重占比计算当前模板应分配的IP数量
		percentage := float64(template.Weight) / float64(totalWeight)
		count := int(float64(len(cr.IpList)) * percentage)

		// 最后一个模板分配剩余所有IP，处理权重计算的舍入误差
		if i == len(shuffledTemplates)-1 {
			count = remainingIPs
		} else {
			// 避免单次分配数量超过剩余IP数
			if count > remainingIPs {
				count = remainingIPs
			}
		}

		ipCountPerTemplate[template.HostTemplateID] = count
		remainingIPs -= count
		remainingTemplates--
	}

	// 按IP列表原始顺序，将IP分配至对应主机模板
	var data []IpInfo
	ipIndex := 0 // IP列表的遍历索引
	for _, template := range shuffledTemplates {
		count := ipCountPerTemplate[template.HostTemplateID]
		// 按计算的数量为当前模板分配IP，保持IP原始顺序
		for i := 0; i < count && ipIndex < len(cr.IpList); i++ {
			data = append(data, IpInfo{
				Ip:             cr.IpList[ipIndex],
				HostTemplateID: template.HostTemplateID,
			})
			ipIndex++
		}
	}

	// 返回IP与主机模板的分配结果
	response.OkWithData(data, c)
}
