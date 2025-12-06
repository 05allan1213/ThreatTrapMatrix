package api

// File: matrix_server/api/remove_deploy.go
// Description: 实现子网IP部署删除API接口

import (
	"errors"
	"matrix_server/internal/global"
	"matrix_server/internal/middleware"
	"matrix_server/internal/models"
	"matrix_server/internal/service/mq_service"
	"matrix_server/internal/service/redis_service/net_lock"
	"matrix_server/internal/service/redis_service/net_progress"
	"matrix_server/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RemoveDeployRequest 删除部署接口的请求参数结构体
type RemoveDeployRequest struct {
	IpList []string `json:"ipList" binding:"required,dive,ip"` // 待删除部署的IP列表
	NetID  uint     `json:"netID" binding:"required"`          // 子网ID
}

// RemoveDeployView 删除部署接口处理函数
func (Api) RemoveDeployView(c *gin.Context) {
	// 绑定并解析请求参数到RemoveDeployRequest结构体
	cr := middleware.GetBind[RemoveDeployRequest](c)
	// 校验IP列表是否为空
	if len(cr.IpList) == 0 {
		response.FailWithMsg("需要选择一个ip进行删除部署", c)
		return
	}

	// 查询子网信息并预加载关联的节点信息
	var model models.NetModel
	err := global.DB.Preload("NodeModel").Take(&model, cr.NetID).Error
	if err != nil {
		response.FailWithMsg("子网不存在", c)
		return
	}
	// 1. 校验节点在线状态
	node := model.NodeModel
	if node.Status != 1 {
		response.FailWithMsg("节点离线", c)
		return
	}

	// 查询子网下指定IP且状态为已部署/部署中的蜜罐IP记录
	var honeyIpList []models.HoneyIpModel
	global.DB.Find(&honeyIpList, "net_id = ? and ip in ? and status in ?", cr.NetID, cr.IpList, []int8{2, 3})
	// 校验所有请求的IP均为已部署状态
	if len(honeyIpList) != len(cr.IpList) {
		response.FailWithMsg("存在未部署的ip", c)
		return
	}

	// 获取上下文日志实例及日志ID
	log := middleware.GetLog(c)
	logID := log.Data["logID"].(string)
	// 组装MQ批量删除部署请求数据
	var batchDeployData = mq_service.BatchRemoveDeployRequest{
		NetID: cr.NetID,
		LogID: logID,
		TanIp: model.IP,
	}

	// 2. 组装IP列表并写入Redis记录部署状态
	for _, ipModel := range honeyIpList {
		batchDeployData.IPList = append(batchDeployData.IPList, mq_service.RemoveDeployIp{
			Ip:       ipModel.IP,
			LinkName: ipModel.Network,
		})
	}

	err = net_lock.Lock(cr.NetID)
	if err != nil {
		response.FailWithMsg("当前子网正在部署中", c)
		return
	}

	// 3. 事务处理：更新IP状态并下发MQ删除部署消息
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		// 将蜜罐IP状态更新为删除中（状态4）
		err = global.DB.Model(&honeyIpList).Update("status", 4).Error
		if err != nil {
			return errors.New("批量部署失败")
		}
		// 记录批量删除部署的IP数量日志
		logrus.Infof("批量删除部署%d诱捕ip", len(honeyIpList))
		// 创建Redis中该子网的部署状态记录
		err = net_progress.Set(cr.NetID, net_progress.NetDeployInfo{
			Type:     3,
			AllCount: int64(len(batchDeployData.IPList)),
		})
		if err != nil {
			logrus.Errorf("设置操作进度失败 %s", err)
			return errors.New("设置操作进度失败")
		}
		// 向MQ下发批量删除部署请求消息
		err = mq_service.SendBatchRemoveDeployMsg(node.Uid, batchDeployData)
		if err != nil {
			return errors.New("部署消息下发失败")
		}
		return nil
	})
	if err != nil {
		// 记录部署失败日志
		logrus.Errorf("部署失败 %s", err)
		response.FailWithError(err, c)
		net_lock.UnLock(cr.NetID)
		return
	}

	// 响应前端删除部署请求提交成功
	// 优化点：若IP数量过多，需拆分下发MQ消息
	response.OkWithMsg("批量删除部署成功，正在删除中", c)
	return
}
