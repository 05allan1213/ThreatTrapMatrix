package net_progress

// File: net_progress.go
// Description: 子网部署进度管理模块，基于Redis实现子网部署/更新/删除操作的进度存储与查询，支持进度数据的序列化/反序列化，记录操作类型、总数、完成数、错误数及错误IP列表，便于实时追踪子网操作进度

import (
	"context"
	"encoding/json"
	"fmt"
	"honey_server/internal/global"
)

// ErrorIp 部署失败的IP信息结构体
type ErrorIp struct {
	Ip  string // 部署失败的IP地址
	Msg string // 失败原因描述
}

// NetDeployInfo 子网部署进度信息结构体
type NetDeployInfo struct {
	Type           int8      // 操作类型：1-部署 2-更新 3-删除
	AllCount       int64     // 操作的总IP数量
	CompletedCount int64     // 已完成的IP数量
	ErrorCount     int64     // 操作失败的IP数量
	ErrorIpList    []ErrorIp // 操作失败的IP详情列表
}

// MarshalBinary 实现encoding.BinaryMarshaler接口
func (n NetDeployInfo) MarshalBinary() (data []byte, err error) {
	return json.Marshal(n)
}

// UnmarshalBinary 实现encoding.BinaryUnmarshaler接口
func (n *NetDeployInfo) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, n)
}

// Set 存储子网部署进度信息到Redis
func Set(netID uint, data NetDeployInfo) error {
	// 构建Redis Key：net_deploy_${netID}，确保子网唯一性
	key := fmt.Sprintf("net_deploy_%d", netID)
	// 将进度信息存入Redis，使用默认上下文，永不过期（-2）
	err := global.Redis.Set(context.Background(), key, data, -2).Err()
	return err
}

// Get 从Redis查询子网部署进度信息
func Get(netID uint) (data NetDeployInfo, err error) {
	// 构建Redis Key：net_deploy_${netID}
	key := fmt.Sprintf("net_deploy_%d", netID)
	// 从Redis读取数据并自动反序列化到NetDeployInfo结构体
	err = global.Redis.Get(context.Background(), key).Scan(&data)
	return
}
