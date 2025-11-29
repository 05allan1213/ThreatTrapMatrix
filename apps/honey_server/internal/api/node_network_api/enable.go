package node_network_api

// File: honey_server/api/node_network_api/enable.go
// Description: 节点网卡启用API接口

import (
	"fmt"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EnableView 处理网卡启用的HTTP请求
func (n *NodeNetworkApi) EnableView(c *gin.Context) {
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
	n.mutex.Lock()
	defer n.mutex.Unlock()
	// 检查网卡当前状态，已启用则直接返回错误
	if model.Status != 1 {
		response.FailWithMsg("网卡已启用，请勿重复启用", c)
		return
	}

	// 使用数据库事务执行启用逻辑，保证数据一致性
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		// 构建网络表记录，关联节点和网卡信息
		var net = models.NetModel{
			NodeID:  model.NodeID,
			Title:   fmt.Sprintf("%s_%s_网络", model.NodeModel.Title, model.Network),
			Network: model.Network,
			IP:      model.IP,
			Mask:    model.Mask,
			Gateway: model.Gateway,
		}
		// 插入网络表记录
		err = tx.Create(&net).Error
		if err != nil {
			return err
		}

		// 构建主机表记录，关联节点和网络信息
		var host = models.HostModel{
			NodeID: model.NodeID,
			NetID:  net.ID,
			IP:     net.IP,
			// Mac字段
			// Manuf字段
		}
		// 插入主机表记录
		err = tx.Create(&host).Error
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
