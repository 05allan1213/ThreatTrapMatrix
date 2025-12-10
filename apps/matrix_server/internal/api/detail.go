package api

// File: matrix_server/api/detail.go
// Description: 子网IP详情查询API接口

import (
	"matrix_server/internal/global"
	"matrix_server/internal/middleware"
	"matrix_server/internal/models"
	"matrix_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// DetailRequest IP详情查询的请求参数结构体
type DetailRequest struct {
	NetID uint   `form:"netID" binding:"required"` // 子网ID
	Ip    string `form:"ip" binding:"required,ip"` // 待查询的IP地址
}

// DetailResponse IP详情查询的响应结构体
// 封装IP类型及对应信息，根据IP类型返回资产/诱捕IP的专属信息
type DetailResponse struct {
	Type      int8       `json:"type"`                // IP类型：1表示资产IP，2表示诱捕IP
	Ip        string     `json:"ip"`                  // 待查询的IP地址
	HostInfo  *HostInfo  `json:"hostInfo,omitempty"`  // 主机信息
	HoneyInfo *HoneyInfo `json:"honeyInfo,omitempty"` // 诱捕IP信息
}

// HostInfo 主机详情信息结构体
type HostInfo struct {
	Mac   string `json:"mac"`   // 主机对应的MAC地址
	Manuf string `json:"manuf"` // 主机对应的设备厂商名称
}

// HoneyInfo 诱捕IP的详情信息结构体
type HoneyInfo struct {
	Mac            string     `json:"mac"`            // 诱捕IP对应的MAC地址
	Status         int8       `json:"status"`         // 诱捕IP的部署状态（1部署中/2部署成功/3部署失败/4删除中）
	HostTemplateID *uint      `json:"hostTemplateID"` // 诱捕IP关联的主机模板ID
	PortList       []PortInfo `json:"portList"`       // 诱捕IP关联的端口转发信息列表
}

// PortInfo 诱捕IP的端口详情信息结构体
type PortInfo struct {
	ServiceID   uint   `json:"serviceID"`   // 端口关联的虚拟服务ID
	ServiceName string `json:"serviceName"` // 端口关联的虚拟服务名称
	Port        int    `json:"port"`        // 端口号
}

// DetailView IP详情查询接口处理函数
func (Api) DetailView(c *gin.Context) {
	// 绑定并解析请求参数到DetailRequest结构体
	cr := middleware.GetBind[DetailRequest](c)

	// 初始化响应数据，填充待查询的IP地址
	data := DetailResponse{
		Ip: cr.Ip,
	}

	// 优先查询诱捕IP记录（预加载关联的端口及服务信息）
	var honeyIp models.HoneyIpModel
	err := global.DB.Preload("PortList.ServiceModel").Take(&honeyIp, "net_id = ? and ip = ?", cr.NetID, cr.Ip).Error
	if err != nil {
		// 诱捕IP不存在时，查询资产IP记录
		data.Type = 1 // 标记IP类型为资产IP
		var hostModel models.HostModel
		err = global.DB.Take(&hostModel, "net_id = ? and ip = ?", cr.NetID, cr.Ip).Error
		if err != nil {
			// 资产IP也不存在时返回错误提示
			response.FailWithMsg("此ip既不是诱捕ip，也不是资产ip", c)
			return
		}
		// 组装资产IP的详情信息
		data.HostInfo = &HostInfo{
			Mac:   hostModel.Mac,
			Manuf: hostModel.Manuf,
		}
		// 返回资产IP详情响应
		response.OkWithData(data, c)
		return
	}

	// 诱捕IP存在时，组装诱捕IP详情响应
	data.Type = 2 // 标记IP类型为诱捕IP
	data.HoneyInfo = &HoneyInfo{
		Mac:            honeyIp.Mac,
		Status:         honeyIp.Status,
		HostTemplateID: honeyIp.HostTemplateID,
	}
	// 遍历诱捕IP关联的端口列表，组装端口详情信息
	for _, model := range honeyIp.PortList {
		data.HoneyInfo.PortList = append(data.HoneyInfo.PortList, PortInfo{
			ServiceID:   model.ServiceID,
			ServiceName: model.ServiceModel.Title,
			Port:        model.Port,
		})
	}

	// 返回诱捕IP详情响应
	response.OkWithData(data, c)
}
