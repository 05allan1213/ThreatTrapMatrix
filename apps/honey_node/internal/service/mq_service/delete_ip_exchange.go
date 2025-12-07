package mq_service

// File: honey_node/service/mq_service/delete_ip_exchange.go
// Description: 删除诱捕IP的MQ消息消费处理逻辑，执行虚拟网络接口删除命令并通过gRPC上报删除状态

import (
	"context"
	"encoding/json"
	"fmt"
	"honey_node/internal/global"
	"honey_node/internal/rpc/node_rpc"
	"honey_node/internal/utils/cmd"

	"github.com/sirupsen/logrus"
)

// DeleteIPRequest 删除诱捕IP的消息结构体（节点）
type DeleteIPRequest struct {
	IpList []IpInfo `json:"ipList"` // 待删除的诱捕IP信息列表
	NetID  uint     `json:"netID"`  // 网络ID
	LogID  string   `json:"logID"`  // 操作日志ID（用于全链路追踪）
}

// IpInfo 单个诱捕IP的删除信息结构体
type IpInfo struct {
	HoneyIPID uint   `json:"honeyIpID"` // 诱捕ipID（关联数据库主键）
	IP        string `json:"ip"`        // 待删除的诱捕IP地址
	Network   string `json:"network"`   // 对应的虚拟网络接口名称（如hy_123）
	IsTan     bool   `json:"isTan"`     // 是否是探针ip
}

// DeleteIpExChange 处理删除诱捕IP的MQ消息，执行虚拟接口删除命令并上报删除状态
func DeleteIpExChange(msg string) error {
	var req DeleteIPRequest
	// 解析MQ消息内容为DeleteIPRequest结构体
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		logrus.Errorf("JSON解析失败: %v, 消息: %s", err, msg)
		return nil // 解析失败返回nil，避免消息重复投递
	}

	// 记录删除操作开始日志（包含请求详情，便于调试）
	global.Log.WithFields(logrus.Fields{
		"req": req,
	}).Infof("删除诱捕ip")

	// 收集待上报的诱捕IPID列表（用于通知服务端删除数据库记录）
	var idList []uint32
	var linkNameList []string
	for _, info := range req.IpList {
		// 执行系统命令删除对应的虚拟网络接口（如hy_123）
		if !info.IsTan {
			cmd.Cmd(fmt.Sprintf("ip link del %s", info.Network))
			linkNameList = append(linkNameList, info.Network)
		} else {
			logrus.Infof("这是探针 %v", info)
		}
		// 将HoneyIPID转换为gRPC要求的uint32类型并加入列表
		idList = append(idList, uint32(info.HoneyIPID))
	}

	// 删除数据库中的数据
	if len(linkNameList) > 0 {
		global.DB.Delete(&linkNameList)
	}

	// 上报删除状态到服务端（通知服务端删除数据库记录）
	reportDeleteIPStatus(int64(req.NetID), idList)
	return nil
}

// reportDeleteIPStatus 通过gRPC向服务端上报IP删除状态，触发数据库记录删除
func reportDeleteIPStatus(netID int64, honeyIPIDList []uint32) error {
	// 调用服务端gRPC接口，上报删除的IPID列表
	response, err := global.GrpcClient.StatusDeleteIP(context.Background(), &node_rpc.StatusDeleteIPRequest{
		HoneyIPIDList: honeyIPIDList,
		NetID:         netID,
	})

	if err != nil {
		logrus.Errorf("上报管理状态失败: %v", err)
		return err // 返回错误表示上报失败（触发MQ重新投递）
	}

	// 记录上报成功日志（包含删除的IPID列表，便于追踪）
	global.Log.WithFields(logrus.Fields{
		"honeyIPIDList": honeyIPIDList,
	}).Infof("上报管理状态成功: %v", response)

	return nil
}
