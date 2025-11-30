package node_network_api

// File: honey_server/api/node_network_api/enable.go
// Description: 节点网卡启用API接口

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/ip"
	"honey_server/internal/utils/response"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 互斥锁，用于控制并发访问
var mutex sync.Mutex

// EnableView 处理网卡启用的HTTP请求
func (NodeNetworkApi) EnableView(c *gin.Context) {
	// 从请求中绑定并获取ID参数
	cr := middleware.GetBind[models.IDRequest](c)

	var model models.NodeNetworkModel
	// 查询指定ID的网卡信息，并预加载关联的节点模型
	err := global.DB.Preload("NodeModel").Take(&model, cr.Id).Error
	if err != nil {
		response.FailWithMsg("网卡不存在", c)
		return
	}

	// 加锁确保并发安全，防止重复操作
	mutex.Lock()
	defer mutex.Unlock()
	// 检查网卡当前状态，已启用则直接返回错误
	if model.Status == 1 {
		response.FailWithMsg("网卡已启用，请勿重复启用", c)
		return
	}

	// 使用数据库事务执行启用逻辑，保证数据一致性
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		ipRange, err1 := ip.ParseCIDRGetUseIPRange(fmt.Sprintf("%s/%d", model.IP, model.Mask))
		if err1 != nil {
			return err1
		}
		// 构建网络表记录，关联节点和网卡信息
		var net = models.NetModel{
			NodeID:             model.NodeID,
			Title:              fmt.Sprintf("%s_%s_网络", model.NodeModel.Title, model.Network),
			Network:            model.Network,
			IP:                 model.IP,
			Mask:               model.Mask,
			Gateway:            model.Gateway,
			CanUseHoneyIPRange: ipRange,
		}
		// 插入网络表记录
		err = tx.Create(&net).Error
		if err != nil {
			return err
		}

		// 更新网卡状态为已启用
		err = tx.Model(&model).Update("status", 1).Error
		return err
	})
	if err != nil {
		logrus.Errorf("网卡启用失败 %s", err)
		response.FailWithMsg("网卡启用失败", c)
		return
	}
	response.OkWithMsg("网卡启用成功", c)
}
