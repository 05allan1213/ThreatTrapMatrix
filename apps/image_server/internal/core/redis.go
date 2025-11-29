package core

// File: image_server/core/redis.go
// Description: Redis客户端初始化模块，提供单例Redis客户端的创建与获取功能

import (
	"image_server/internal/global"
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// client 全局Redis客户端实例，通过单例模式初始化
var redisClient *redis.Client

// InitRedis 初始化Redis客户端，建立连接并验证
func InitRedis() (client *redis.Client) {
	// 获取全局配置中的Redis配置信息
	conf := global.Config.Redis
	// 创建Redis客户端实例
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.Addr,     // Redis服务地址
		Password: conf.Password, // Redis访问密码
		DB:       conf.DB,       // 使用的Redis数据库编号
	})

	// 发送Ping命令验证连接
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		logrus.Fatalf("连接redis失败 %s", err)
		return
	}
	logrus.Infof("成功连接redis")
	return rdb
}

// onceRedis 用于确保Redis客户端仅初始化一次的同步控制
var onceRedis sync.Once

// GetRedisClient 获取单例Redis客户端实例（懒加载）
func GetRedisClient() *redis.Client {
	onceRedis.Do(func() {
		redisClient = InitRedis()
	})
	return redisClient
}
