package net_progress

// File: matrix_server/service/net_progress/enter.go
// Description: 子网部署进度管理模块，基于Redis实现部署进度信息的序列化存储与读取，包含部署总数、完成数、错误数及错误IP记录等核心数据的管理

import (
	"context"
	"encoding/json"
	"fmt"
	"matrix_server/internal/global"
)

// ErrorIp 部署失败的IP信息结构体
type ErrorIp struct {
	Ip  string // 部署失败的IP地址
	Msg string // 部署失败的错误信息
}

// NetDeployInfo 子网部署进度信息结构体
type NetDeployInfo struct {
	Type           int8      // 部署类型标识（1表示批量部署）
	AllCount       int64     // 部署任务的总IP数量
	CompletedCount int64     // 已完成部署的IP数量
	ErrorCount     int64     // 部署失败的IP数量
	ErrorIpList    []ErrorIp // 部署失败的IP列表（含错误信息）
}

// MarshalBinary 实现BinaryMarshaler接口
func (n NetDeployInfo) MarshalBinary() (data []byte, err error) {
	return json.Marshal(n)
}

// UnmarshalBinary 实现BinaryUnmarshaler接口
func (n *NetDeployInfo) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, n)
}

// Set 存储子网部署进度信息到Redis
func Set(netID uint, data NetDeployInfo) error {
	// 构建Redis存储Key（格式：net_deploy_子网ID）
	key := fmt.Sprintf("net_deploy_%d", netID)
	// 存入Redis，过期时间-2表示使用Redis默认配置（永不过期）
	err := global.Redis.Set(context.Background(), key, data, -2).Err()
	return err
}

// Get 从Redis读取子网部署进度信息
func Get(netID uint) (data NetDeployInfo, err error) {
	// 构建Redis读取Key（格式：net_deploy_子网ID）
	key := fmt.Sprintf("net_deploy_%d", netID)
	// 从Redis读取数据并解析到NetDeployInfo结构体
	err = global.Redis.Get(context.Background(), key).Scan(&data)
	return
}
