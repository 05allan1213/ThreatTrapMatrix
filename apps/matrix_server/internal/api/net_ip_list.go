package api

// File: matrix_server/api/net_ip_list.go
// Description: 子网IP列表查询API接口

import (
	"context"
	"fmt"
	"matrix_server/internal/global"
	"matrix_server/internal/middleware"
	"matrix_server/internal/models"
	"matrix_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// NetIpListRequest 子网IP列表查询请求参数结构体
type NetIpListRequest struct {
	models.PageInfo      // 分页基础参数
	NetID           uint `form:"netID" binding:"required"` // 子网ID
}

// NetIpInfo 子网IP信息结构体
type NetIpInfo struct {
	Ip   string `json:"ip"`   // IP地址
	Type int8   `json:"type"` // IP状态类型：
	// 0 - 空闲IP
	// 1 - 主机IP（资产IP）
	// 2 - 诱捕IP
	// 3 - 部署中IP
	// 4 - 删除中IP
	// 5 - 操作失败的IP
}

// NetIpListResponse 子网IP列表查询响应结构体
type NetIpListResponse struct {
	IpCount            int         `json:"ipCount"`            // 子网内IP总数
	AssetCount         int         `json:"assetCount"`         // 子网内资产IP（主机IP）总数
	IdleCount          int         `json:"idleCount"`          // 子网内空闲IP总数
	HoneyIpCount       int         `json:"honeyIpCount"`       // 子网内诱捕IP总数
	CanUseHoneyIPRange string      `json:"canUseHoneyIPRange"` // 子网可用的诱捕IP范围
	TotalPages         int         `json:"totalPages"`         // 分页总页数
	Title              string      `json:"title"`              // 子网名称
	Subnet             string      `json:"subnet"`             // 子网网段
	IsAction           bool        `json:"isAction"`           // 当前子网是否正在执行操作
	List               []NetIpInfo `json:"list"`               // 分页后的IP列表数据
}

// NetIpListView 子网IP列表查询接口处理函数
func (Api) NetIpListView(c *gin.Context) {
	// 绑定并解析查询请求参数
	cr := middleware.GetBind[NetIpListRequest](c)

	// 查询指定ID的子网基础信息
	var model models.NetModel
	err := global.DB.Take(&model, cr.NetID).Error
	if err != nil {
		response.FailWithMsg("子网不存在", c)
		return
	}

	// 计算该子网的IP地址范围
	ipRange, err := model.IpRange()
	if err != nil {
		response.FailWithMsg("计算ip范围失败", c)
		return
	}

	// 初始化响应数据结构体，用于存储各类IP统计数据
	var data = NetIpListResponse{}

	// 查询子网下已存在的资产IP，构建IP映射表用于状态判断
	var hostMap = map[string]bool{}
	var assetsList []models.HostModel
	global.DB.Find(&assetsList, "net_id = ?", cr.NetID)
	data.AssetCount = len(assetsList)
	for _, hostModel := range assetsList {
		hostMap[hostModel.IP] = true
	}

	// 查询子网下已存在的诱捕IP，构建IP映射表用于状态判断
	var honeIpMap = map[string]models.HoneyIpModel{}
	var honeyIpList []models.HoneyIpModel
	global.DB.Find(&honeyIpList, "net_id = ?", cr.NetID)
	data.HoneyIpCount = len(honeyIpList)
	for _, honeyModel := range honeyIpList {
		honeIpMap[honeyModel.IP] = honeyModel
	}

	// 计算各类IP总数（与分页无关）
	data.IpCount = len(ipRange)                                           // 子网总IP数
	data.IdleCount = data.IpCount - (data.AssetCount + data.HoneyIpCount) // 空闲IP数=总IP数-资产IP数-诱捕IP数
	data.CanUseHoneyIPRange = model.CanUseHoneyIPRange                    // 赋值子网可用诱捕IP范围

	// 分页参数处理：设置默认值
	if cr.Page <= 0 {
		cr.Page = 1 // 页码默认值为1
	}
	if cr.Limit <= 0 || cr.Limit > 254 {
		// 根据子网掩码调整每页最大条数
		if model.Mask < 24 {
			cr.Limit = 255
		} else {
			cr.Limit = 254
		}
	}

	// 计算分页截取的起始和结束索引
	startIndex := (cr.Page - 1) * cr.Limit
	endIndex := startIndex + cr.Limit
	// 边界值处理：防止索引超出IP范围长度
	if startIndex > len(ipRange) {
		startIndex = len(ipRange)
	}
	if endIndex > len(ipRange) {
		endIndex = len(ipRange)
	}
	// 截取分页后的IP列表
	paginatedIPs := ipRange[startIndex:endIndex]

	// 计算分页总页数（向上取整）
	totalPages := (len(ipRange) + cr.Limit - 1) / cr.Limit

	// 从Redis获取当前子网下部署中的IP状态，构建映射表用于状态判断
	key := fmt.Sprintf("deploy_create_%d", cr.NetID)
	maps := global.Redis.HGetAll(context.Background(), key).Val()
	var creatingMap = map[string]bool{}
	for k, s2 := range maps {
		if s2 == "1" {
			creatingMap[k] = true // 标记该IP处于部署中状态
		}
	}

	// 构建分页后的IP信息列表（包含状态类型）
	var list = make([]NetIpInfo, 0)
	for _, p := range paginatedIPs {
		info := NetIpInfo{
			Ip: p,
		}
		// 判断IP类型：资产IP（主机IP）
		if hostMap[p] {
			info.Type = 1
		}
		// 判断IP类型：诱捕IP
		honeyIpModel, ok := honeIpMap[p]
		if ok {
			if honeyIpModel.Status == 1 { // 创建中
				info.Type = 3
			}
			if honeyIpModel.Status == 4 { // 删除中
				info.Type = 4
			}
			if honeyIpModel.Status == 3 { // 失败的
				info.Type = 5
			}
			if honeyIpModel.Status == 2 { // 运行中
				info.Type = 2
			}
		}
		// 判断IP类型：部署中IP
		if creatingMap[p] {
			info.Type = 3
		}
		// 待补充逻辑：判断IP是否为删除中/操作失败状态
		list = append(list, info)
	}

	// 填充响应数据
	data.List = list
	data.TotalPages = totalPages
	data.Title = model.Title     // 子网名称
	data.Subnet = model.Subnet() // 子网网段

	// 判断当前子网是否正在执行操作
	err = global.Redis.Get(context.Background(), fmt.Sprintf("deploy_action_lock_%d", cr.NetID)).Err()
	if err == nil {
		data.IsAction = true
	}

	// 返回成功响应及查询结果
	response.OkWithData(data, c)
}
